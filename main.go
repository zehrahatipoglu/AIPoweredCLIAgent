package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/invopop/jsonschema"
	openai "github.com/sashabaranov/go-openai"
)

type Agent struct {
	client         *openai.Client
	getUserMessage func() (string, bool)
	tools          []ToolDefinition
}

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Lütfen OPENAI_API_KEY ortam değişkenini ayarlayın.")
		return
	}

	client := openai.NewClient(apiKey)

	scanner := bufio.NewScanner(os.Stdin)
	getUserMessage := func() (string, bool) {
		if !scanner.Scan() {
			return "", false
		}
		return scanner.Text(), true
	}

	tools := []ToolDefinition{
		ReadFileDefinition,
		ListFilesDefinition,
		EditFileDefinition,
	}

	agent := NewAgent(client, getUserMessage, tools)
	if err := agent.Run(context.TODO()); err != nil {
		fmt.Printf("Error: %s\n", err)
	}
}

func NewAgent(client *openai.Client, getUserMessage func() (string, bool), tools []ToolDefinition) *Agent {
	return &Agent{
		client:         client,
		getUserMessage: getUserMessage,
		tools:          tools,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	fmt.Println("Chat with ChatGPT (use Ctrl+C to quit)")

	var messages []openai.ChatCompletionMessage

	for {
		// Kullanıcı mesajı al
		fmt.Print("\u001b[94mYou\u001b[0m: ")
		userInput, ok := a.getUserMessage()
		if !ok {
			break
		}

		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: userInput,
		})

		// Tool destekli yanıt al
		resp, err := a.runInference(ctx, messages)
		if err != nil {
			return err
		}

		msg := resp.Choices[0].Message
		if msg.FunctionCall != nil {
			// ✅ Tool çağrısı varsa çalıştır
			toolOutput, err := a.executeTool(*msg.FunctionCall)
			if err != nil {
				return err
			}

			// Tool çağrısını mesajlara ekle
			messages = append(messages, msg) // FunctionCall içeren assistant mesajı
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleFunction,
				Name:    msg.FunctionCall.Name,
				Content: toolOutput,
			})

			// Tool çıktılarına göre modelden tekrar yanıt al
			resp, err := a.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
				Model:    openai.GPT3Dot5Turbo,
				Messages: messages,
			})
			if err != nil {
				return err
			}

			msg = resp.Choices[0].Message
			messages = append(messages, msg)
		} else {
			messages = append(messages, msg)
		}

		fmt.Printf("\u001b[93mChatGPT\u001b[0m: %s\n", msg.Content)
	}

	return nil
}

func (a *Agent) executeTool(fc openai.FunctionCall) (string, error) {
	var toolDef ToolDefinition
	found := false

	for _, tool := range a.tools {
		if tool.Name == fc.Name {
			toolDef = tool
			found = true
			break
		}
	}

	if !found {
		return "", fmt.Errorf("Tool not found: %s", fc.Name)
	}

	fmt.Printf("\u001b[92mtool\u001b[0m: %s(%s)\n", fc.Name, fc.Arguments)

	output, err := toolDef.Function([]byte(fc.Arguments))
	if err != nil {
		return "", err
	}

	return output, nil
}

type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]interface{} // JSON Schema
	Function    func(input json.RawMessage) (string, error)
}

func (a *Agent) runInference(ctx context.Context, conversation []openai.ChatCompletionMessage) (*openai.ChatCompletionResponse, error) {
	// ToolDefinition listesinden OpenAI FunctionDefinition üret
	var openaiTools []openai.FunctionDefinition
	for _, tool := range a.tools {
		openaiTools = append(openaiTools, openai.FunctionDefinition{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Parameters,
		})
	}

	req := openai.ChatCompletionRequest{
		Model:        openai.GPT3Dot5Turbo, // veya GPT3Dot5Turbo
		Messages:     conversation,
		Functions:    openaiTools,
		FunctionCall: "auto", // veya tool.Name
		MaxTokens:    1024,
	}

	resp, err := a.client.CreateChatCompletion(ctx, req)
	return &resp, err
}

type ReadFileInput struct {
	Path string `json:"path" jsonschema_description:"The relative path of a file in the working directory."`
}

var ReadFileDefinition = ToolDefinition{
	Name:        "read_file",
	Description: "Read the contents of a given relative file path. Use this when you want to see what's inside a file. Do not use this with directory names.",
	Parameters:  GenerateSchema[ReadFileInput](), // map[string]interface{}
	Function:    ReadFile,
}

func ReadFile(input json.RawMessage) (string, error) {
	var readFileInput ReadFileInput
	if err := json.Unmarshal(input, &readFileInput); err != nil {
		return "", err
	}

	content, err := os.ReadFile(readFileInput.Path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func GenerateSchema[T any]() map[string]interface{} {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)

	// JSON'a çevirip sonra geri map'e döndür
	data, _ := json.Marshal(schema)
	var result map[string]interface{}
	_ = json.Unmarshal(data, &result)

	return result
}

var ListFilesDefinition = ToolDefinition{
	Name:        "list_files",
	Description: "List files and directories at a given path. If no path is provided, lists files in the current directory.",
	Parameters:  GenerateSchema[ListFilesInput](),
	Function:    ListFiles,
}

type ListFilesInput struct {
	Path string `json:"path,omitempty" jsonschema_description:"Optional relative path to list files from. Defaults to current directory if not provided."`
}

func ListFiles(input json.RawMessage) (string, error) {
	var listFilesInput ListFilesInput
	if err := json.Unmarshal(input, &listFilesInput); err != nil {
		return "", err
	}

	dir := "."
	if listFilesInput.Path != "" {
		dir = listFilesInput.Path
	}

	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		if relPath != "." {
			if info.IsDir() {
				files = append(files, relPath+"/")
			} else {
				files = append(files, relPath)
			}
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	result, err := json.Marshal(files)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

type EditFileInput struct {
	Path   string `json:"path" jsonschema_description:"The path to the file"`
	OldStr string `json:"old_str" jsonschema_description:"Text to search for - must match exactly and must only have one match exactly"`
	NewStr string `json:"new_str" jsonschema_description:"Text to replace old_str with"`
}

var EditFileDefinition = ToolDefinition{
	Name: "edit_file",
	Description: `Make edits to a text file.

Replaces 'old_str' with 'new_str' in the given file. 'old_str' and 'new_str' MUST be different from each other.

If the file specified with path doesn't exist, it will be created.`,
	Parameters: GenerateSchema[EditFileInput](), // OpenAI için InputSchema değil!
	Function:   EditFile,
}

func EditFile(input json.RawMessage) (string, error) {
	var editFileInput EditFileInput
	if err := json.Unmarshal(input, &editFileInput); err != nil {
		return "", fmt.Errorf("invalid JSON input: %w", err)
	}

	// Geçersiz parametre kontrolü
	if editFileInput.Path == "" || editFileInput.OldStr == editFileInput.NewStr {
		return "", fmt.Errorf("invalid input: path is empty or old_str == new_str")
	}

	// Dosyayı oku
	content, err := os.ReadFile(editFileInput.Path)
	if err != nil {
		// Dosya yoksa
		if os.IsNotExist(err) {
			// Eğer old_str boşsa, doğrudan yeni dosya oluştur
			if editFileInput.OldStr == "" {
				return createNewFile(editFileInput.Path, editFileInput.NewStr)
			}

			// Eğer old_str boş değilse, yine de dosyayı baştan yarat ama old_str'yi dahil ederek
			initialContent := editFileInput.OldStr + "\n" + editFileInput.NewStr
			return createNewFile(editFileInput.Path, initialContent)
		}

		// Diğer okuma hataları
		return "", fmt.Errorf("error reading file: %w", err)
	}

	// İçeriği düzenle
	oldContent := string(content)
	newContent := strings.Replace(oldContent, editFileInput.OldStr, editFileInput.NewStr, -1)

	// Değişiklik olmadıysa → hata döndür
	if oldContent == newContent && editFileInput.OldStr != "" {
		return "", fmt.Errorf("old_str not found in file: no changes made")
	}

	// Dosyayı tekrar yaz
	if err := os.WriteFile(editFileInput.Path, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("error writing file: %w", err)
	}

	return "File edited successfully.", nil
}

func createNewFile(filePath, content string) (string, error) {
	dir := path.Dir(filePath)
	if dir != "." {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}
	}

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}

	return fmt.Sprintf("Successfully created file %s", filePath), nil
}
