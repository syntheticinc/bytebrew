package indexing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TC-I-05: Go parsing — functions, methods, structs extracted
func TestChunker_Go(t *testing.T) {
	chunker := NewChunker()

	content := `package main

import "fmt"

type UserService struct {
	repo UserRepo
}

func NewUserService(repo UserRepo) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetByID(id string) (*User, error) {
	return s.repo.Find(id)
}

type UserRepo interface {
	Find(id string) (*User, error)
	Save(u *User) error
}
`

	chunks := chunker.ChunkFile("/project/main.go", content, "go")
	require.NotEmpty(t, chunks)

	names := extractNames(chunks)
	assert.Contains(t, names, "UserService")
	assert.Contains(t, names, "NewUserService")
	assert.Contains(t, names, "GetByID")
	assert.Contains(t, names, "UserRepo")

	for _, chunk := range chunks {
		assert.NotEmpty(t, chunk.ID, "chunk ID should not be empty")
		assert.Equal(t, "/project/main.go", chunk.FilePath)
		assert.Equal(t, "go", chunk.Language)
		assert.True(t, chunk.StartLine > 0)
		assert.True(t, chunk.EndLine >= chunk.StartLine)
	}

	// Verify chunk types
	typeMap := make(map[string]ChunkType)
	for _, c := range chunks {
		typeMap[c.Name] = c.ChunkType
	}
	assert.Equal(t, ChunkStruct, typeMap["UserService"])
	assert.Equal(t, ChunkFunction, typeMap["NewUserService"])
	assert.Equal(t, ChunkMethod, typeMap["GetByID"])
	assert.Equal(t, ChunkInterface, typeMap["UserRepo"])
}

// TC-I-06: TypeScript parsing — functions, classes, interfaces extracted
func TestChunker_TypeScript(t *testing.T) {
	chunker := NewChunker()

	content := `export class UserController {
  constructor(private service: UserService) {}

  async getUser(id: string) {
    return this.service.getById(id);
  }
}

export async function createApp() {
  const controller = new UserController(new UserService());
  return controller;
}

export const handler = async (req: Request) => {
  return new Response("ok");
};
`

	chunks := chunker.ChunkFile("/project/app.ts", content, "typescript")
	require.NotEmpty(t, chunks)

	names := extractNames(chunks)
	assert.Contains(t, names, "UserController")
	assert.Contains(t, names, "createApp")
	assert.Contains(t, names, "handler")
}

// TC-I-07: Python parsing — functions, classes extracted
func TestChunker_Python(t *testing.T) {
	chunker := NewChunker()

	content := `class UserService:
    def __init__(self, repo):
        self.repo = repo

    def get_by_id(self, user_id: str):
        return self.repo.find(user_id)

async def create_user(name: str):
    service = UserService(repo)
    return service.create(name)
`

	chunks := chunker.ChunkFile("/project/service.py", content, "python")
	require.NotEmpty(t, chunks)

	names := extractNames(chunks)
	assert.Contains(t, names, "UserService")
	assert.Contains(t, names, "create_user")
}

func TestChunker_UnknownLanguage(t *testing.T) {
	chunker := NewChunker()

	content := `some random content
that has no known patterns
but is long enough to be a chunk`

	chunks := chunker.ChunkFile("/project/data.txt", content, "unknown")
	require.Len(t, chunks, 1)
	assert.Equal(t, ChunkOther, chunks[0].ChunkType)
	assert.Equal(t, "file", chunks[0].Name)
}

func TestChunker_EmptyFile(t *testing.T) {
	chunker := NewChunker()
	chunks := chunker.ChunkFile("/project/empty.go", "", "go")
	assert.Empty(t, chunks)
}

func TestChunker_Rust(t *testing.T) {
	chunker := NewChunker()

	content := `pub struct Config {
    pub host: String,
    pub port: u16,
}

pub trait Service {
    fn start(&self) -> Result<(), Error>;
}

impl Config {
    pub fn new() -> Self {
        Config { host: "localhost".into(), port: 8080 }
    }
}

pub async fn run(config: Config) -> Result<(), Error> {
    println!("Running on {}:{}", config.host, config.port);
    Ok(())
}
`

	chunks := chunker.ChunkFile("/project/main.rs", content, "rust")
	require.NotEmpty(t, chunks)

	names := extractNames(chunks)
	assert.Contains(t, names, "Config")
	assert.Contains(t, names, "Service")
	assert.Contains(t, names, "run")
}

func TestGenerateChunkID(t *testing.T) {
	id1 := generateChunkID("/a.go", 1, "main")
	id2 := generateChunkID("/a.go", 1, "main")
	id3 := generateChunkID("/b.go", 1, "main")

	assert.Equal(t, id1, id2, "same input should produce same ID")
	assert.NotEqual(t, id1, id3, "different input should produce different ID")
	assert.Len(t, id1, 16, "ID should be 16 hex chars")
}

func extractNames(chunks []CodeChunk) []string {
	var names []string
	for _, c := range chunks {
		names = append(names, c.Name)
	}
	return names
}
