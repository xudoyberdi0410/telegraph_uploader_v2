package main

import (
	"context"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// DialogProvider interface mocks Wails dialogs
type DialogProvider interface {
	OpenDirectory(ctx context.Context, options wailsRuntime.OpenDialogOptions) (string, error)
	OpenMultipleFiles(ctx context.Context, options wailsRuntime.OpenDialogOptions) ([]string, error)
}

// WailsDialogProvider implements DialogProvider using real Wails runtime
type WailsDialogProvider struct{}

func (w *WailsDialogProvider) OpenDirectory(ctx context.Context, options wailsRuntime.OpenDialogOptions) (string, error) {
	return wailsRuntime.OpenDirectoryDialog(ctx, options)
}

func (w *WailsDialogProvider) OpenMultipleFiles(ctx context.Context, options wailsRuntime.OpenDialogOptions) ([]string, error) {
	return wailsRuntime.OpenMultipleFilesDialog(ctx, options)
}
