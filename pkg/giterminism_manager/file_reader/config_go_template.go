package file_reader

import (
	"context"
	"fmt"
	"path/filepath"
)

func (r FileReader) ConfigGoTemplateFilesGlob(ctx context.Context, pattern string) (map[string]interface{}, error) {
	result := map[string]interface{}{}

	if err := r.configurationFilesGlob(
		ctx,
		pattern,
		r.giterminismConfig.IsUncommittedConfigGoTemplateRenderingFileAccepted,
		func(relPath string, data []byte, err error) error {
			if err != nil {
				return err
			}

			result[filepath.ToSlash(relPath)] = string(data)

			return nil
		},
	); err != nil {
		return nil, fmt.Errorf("{{ .Files.Glob '%s' }}: %s", pattern, err)
	}

	return result, nil
}

func (r FileReader) ConfigGoTemplateFilesGet(ctx context.Context, relPath string) ([]byte, error) {
	if err := r.checkConfigurationFileExistence(ctx, relPath, r.giterminismConfig.IsUncommittedConfigGoTemplateRenderingFileAccepted); err != nil {
		return nil, fmt.Errorf("{{ .Files.Get '%s' }}: %s", relPath, err)
	}

	data, err := r.checkAndReadConfigurationFile(ctx, relPath, r.giterminismConfig.IsUncommittedConfigGoTemplateRenderingFileAccepted)
	if err != nil {
		return nil, fmt.Errorf("{{ .Files.Get '%s' }}: %s", relPath, err)
	}

	return data, nil
}
