package main

import (
	"context"
	"dagger/doublestag/internal/dagger"
	"fmt"
)

var goVersion = "1.24"

type Backend struct{}

func (m *Backend) Lint(ctx context.Context, src *dagger.Directory) (string, error) {
	return dag.Container().
		From("danielmichaels/ci-toolkit").
		WithDirectory("/src", src).
		WithWorkdir("/src").
		WithExec([]string{"task", "betteralign"}, dagger.ContainerWithExecOpts{}).
		WithExec([]string{"task", "golines-ci"}, dagger.ContainerWithExecOpts{}).
		WithExec([]string{"task", "golangci"}, dagger.ContainerWithExecOpts{}).
		Stdout(ctx)
}
func (m *Backend) Test(ctx context.Context, src *dagger.Directory) (string, error) {
	return dag.Container().
		From("danielmichaels/ci-toolkit").
		WithDirectory("/src", src).
		WithWorkdir("/src").
		WithExec([]string{"go", "build", "-v", "./..."}, dagger.ContainerWithExecOpts{}).
		WithExec([]string{"go", "test", "-v", "-race", "./..."}, dagger.ContainerWithExecOpts{}).
		Stdout(ctx)
}

func (m *Backend) Build(
	ctx context.Context,
	src *dagger.Directory,
	dockerfile *dagger.File,
) (*dagger.Container, error) {
	workspace := dag.Container().
		WithDirectory(".", src).
		WithWorkdir(".").
		WithFile("./Dockerfile", dockerfile).
		Directory(".")
	ref := dag.Container().
		Build(workspace, dagger.ContainerBuildOpts{
			Dockerfile: "Dockerfile",
		})
	return ref, nil
}

func (m *Backend) LintTestBuild(
	ctx context.Context,
	src *dagger.Directory,
	dockerfile *dagger.File,
) (*dagger.Container, error) {
	_, err := m.Lint(ctx, src)
	if err != nil {
		return nil, err
	}
	_, err = m.Test(ctx, src)
	if err != nil {
		return nil, err
	}
	return m.Build(ctx, src, dockerfile)
}

func (m *Backend) Publish(
	ctx context.Context,
	buildContext *dagger.Directory,
	dockerfile *dagger.File,
	registry, imageName, registryUsername string,
	registryPassword *dagger.Secret,
) error {
	b, err := m.LintTestBuild(ctx, buildContext, dockerfile)
	if err != nil {
		return err
	}
	_, err = b.WithRegistryAuth(registry, registryUsername, registryPassword).
		Publish(ctx, fmt.Sprintf("%s/%s:testing123", registry, imageName))
	if err != nil {
		return err
	}
	return err
}
