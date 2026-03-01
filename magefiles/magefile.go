package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const templDir = "./internal/templates"

// Dbup runs dbmate to apply db migrations
func Dbup() error {
	if _, err := exec.LookPath("dbmate"); err != nil {
		fmt.Println(">> dbmate not found; install with:")
		fmt.Println("   go install https://github.com/amacneil/dbmate@latest")
		return err
	}
	fmt.Println(">> dbmate up")
	return sh.Run("dbmate", "up")
}

// Generate runs templ generate targeting the templates directory.
// This must be run before Build or Dev any time a .templ file changes.
func Generate() error {
	if _, err := exec.LookPath("templ"); err != nil {
		fmt.Println(">> templ not found; install with:")
		fmt.Println("   go install github.com/a-h/templ/cmd/templ@latest")
		return err
	}
	fmt.Println(">> templ generate", templDir)
	return sh.Run("templ", "generate", templDir)
}

// Build generates templ output, tidies deps, then compiles to ./bin/w2c-server.
func Build() error {
	mg.Deps(Generate, Tidy)
	fmt.Println(">> Building server binary...")
	return sh.Run("go", "build", "-o", "bin/w2c-server", "./cmd/server")
}

// Run builds then executes the binary.
func Run() error {
	mg.Deps(Build)
	fmt.Println(">> Starting server on :8080 ...")
	return sh.Run("./bin/w2c-server")
}

// Dev generates templates then starts the server via go run.
// Use Watch instead if you want live template reloading.
func Dev() error {
	mg.Deps(Generate)
	fmt.Println(">> Dev mode: go run ./cmd/server ...")
	cmd := exec.Command("go", "run", "./cmd/server")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "PORT=8080")
	return cmd.Run()
}

// Watch runs templ generate --watch in the background and the server in the
// foreground. Ctrl-C stops both. Use this for active template development.
func Watch() error {
	if _, err := exec.LookPath("templ"); err != nil {
		fmt.Println(">> templ not found; install with:")
		fmt.Println("   go install github.com/a-h/templ/cmd/templ@latest")
		return err
	}

	// Initial generate before starting the server.
	mg.Deps(Generate)

	fmt.Println(">> Starting templ watcher...")
	watcher := exec.Command("templ", "generate", "--watch", "-f", templDir)
	watcher.Stdout = os.Stdout
	watcher.Stderr = os.Stderr
	if err := watcher.Start(); err != nil {
		return fmt.Errorf("start templ watcher: %w", err)
	}

	fmt.Println(">> Starting server (go run)...")
	server := exec.Command("go", "run", "./cmd/server")
	server.Stdout = os.Stdout
	server.Stderr = os.Stderr
	server.Env = append(os.Environ(), "PORT=8080")
	if err := server.Start(); err != nil {
		watcher.Process.Kill()
		return fmt.Errorf("start server: %w", err)
	}

	// Wait for Ctrl-C then cleanly stop both processes.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n>> Shutting down...")
	server.Process.Kill()
	watcher.Process.Kill()
	return nil
}

// Tidy runs go mod tidy.
func Tidy() error {
	fmt.Println(">> go mod tidy...")
	return sh.Run("go", "mod", "tidy")
}

// Test generates templates then runs all unit tests.
func Test() error {
	mg.Deps(Generate)
	fmt.Println(">> Running tests...")
	return sh.Run("go", "test", "./...")
}

// Lint runs golangci-lint if available.
func Lint() error {
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		fmt.Println(">> golangci-lint not found; skipping.")
		return nil
	}
	return sh.Run("golangci-lint", "run", "./...")
}

// Clean removes build artifacts, generated templ files, and the local SQLite DB.
func Clean() error {
	fmt.Println(">> Cleaning...")
	os.RemoveAll("bin")
	os.Remove("w2c.db")
	// Remove generated _templ.go files
	return sh.Run("find", templDir, "-name", "*_templ.go", "-delete")
}

// Install builds and installs the binary to $GOPATH/bin.
func Install() error {
	mg.Deps(Build)
	return sh.Run("go", "install", "./cmd/server")
}

func init() {
	err := godotenv.Load()
	if err != nil {
		slog.Warn("error loading .env file", "err", err)
	}
}
