package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Build compiles the server binary to ./bin/w2c-server
func Build() error {
	mg.Deps(Tidy)
	fmt.Println(">> Building server binary...")
	return sh.Run("go", "build", "-o", "bin/w2c-server", "./cmd/server")
}

// Run starts the development server (hot-rebuild not included; use Air for that)
func Run() error {
	mg.Deps(Build)
	fmt.Println(">> Starting server on :8080 ...")
	return sh.Run("./bin/w2c-server")
}

// Dev starts the server directly via go run (faster for dev)
func Dev() error {
	fmt.Println(">> Dev mode: go run ./cmd/server ...")
	cmd := exec.Command("go", "run", "./cmd/server")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "PORT=8080")
	return cmd.Run()
}

// Test runs all unit tests
func Test() error {
	fmt.Println(">> Running tests...")
	return sh.Run("go", "test", "./...")
}

// Tidy runs go mod tidy
func Tidy() error {
	fmt.Println(">> go mod tidy...")
	return sh.Run("go", "mod", "tidy")
}

// Clean removes build artifacts and the local SQLite database
func Clean() error {
	fmt.Println(">> Cleaning build artifacts...")
	os.RemoveAll("bin")
	os.Remove("w2c.db")
	return nil
}

// Generate runs templ generate (when .templ files are added)
func Generate() error {
	if _, err := exec.LookPath("templ"); err != nil {
		fmt.Println(">> templ not found; install with: go install github.com/a-h/templ/cmd/templ@latest")
		return err
	}
	fmt.Println(">> Running templ generate...")
	return sh.Run("templ", "generate")
}

// Lint runs golangci-lint if available
func Lint() error {
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		fmt.Println(">> golangci-lint not found; skipping.")
		return nil
	}
	return sh.Run("golangci-lint", "run", "./...")
}

// Install installs the binary to $GOPATH/bin
func Install() error {
	mg.Deps(Build)
	return sh.Run("go", "install", "./cmd/server")
}
