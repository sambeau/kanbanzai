package mcp

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/buildinfo"
	"kanbanzai/internal/install"
)

// ServerInfoTool returns the server_info MCP tool registered in the core group.
// It reports build metadata, binary location, install record, and in-sync status.
func ServerInfoTool() []server.ServerTool {
	return []server.ServerTool{serverInfoTool()}
}

func serverInfoTool() server.ServerTool {
	tool := mcp.NewTool("server_info",
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("Server Build Information"),
		mcp.WithDescription(
			"Get server build and installation metadata. "+
				"Returns version, git SHA, build time, Go version, binary path, "+
				"install record, and whether the running binary matches the install record. "+
				"No input arguments.",
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleServerInfo(".")
	}

	return server.ServerTool{Tool: tool, Handler: handler}
}

func handleServerInfo(root string) (*mcp.CallToolResult, error) {
	gitSHA := buildinfo.GitSHA
	gitSHAShort := deriveGitSHAShort(gitSHA)
	dirty := buildinfo.Dirty == "true"

	binaryPath := ""
	if exePath, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
			binaryPath = resolved
		} else {
			binaryPath = exePath
		}
	}

	result := map[string]any{
		"version":       buildinfo.Version,
		"git_sha":       gitSHA,
		"git_sha_short": gitSHAShort,
		"build_time":    buildinfo.BuildTime,
		"go_version":    runtime.Version(),
		"binary_path":   binaryPath,
		"dirty":         dirty,
	}

	// Read install record fresh on each call (not cached).
	// ReadRecord returns nil, nil when the file does not exist.
	rec, err := install.ReadRecord(root)
	if err != nil {
		return nil, err
	}

	if rec != nil {
		result["install_record"] = map[string]any{
			"git_sha":      rec.GitSHA,
			"installed_at": rec.InstalledAt.Format("2006-01-02T15:04:05Z07:00"),
			"installed_by": rec.InstalledBy,
			"binary_path":  rec.BinaryPath,
		}
	} else {
		result["install_record"] = nil
	}

	result["in_sync"] = deriveInSync(gitSHA, rec)

	return buildResult(result, nil, false), nil
}

// deriveGitSHAShort returns the first 7 characters of the git SHA,
// or "unknown" if the SHA is "unknown".
func deriveGitSHAShort(sha string) string {
	if sha == "unknown" || sha == "" {
		return "unknown"
	}
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// deriveInSync computes the in_sync status:
//   - true if git_sha equals install_record.git_sha and neither is "unknown"
//   - false if both are known and they differ
//   - nil if git_sha is "unknown", or install_record is nil, or install_record.git_sha is empty
func deriveInSync(gitSHA string, rec *install.InstallRecord) any {
	if gitSHA == "unknown" || gitSHA == "" {
		return nil
	}
	if rec == nil {
		return nil
	}
	if rec.GitSHA == "" {
		return nil
	}
	return gitSHA == rec.GitSHA
}
