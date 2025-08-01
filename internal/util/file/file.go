package util

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ZipOptions ZIP 压缩选项
type ZipOptions struct {
	// 压缩级别 (0-9, 0=不压缩, 9=最大压缩)
	CompressionLevel uint16
	// 是否包含隐藏文件
	IncludeHidden bool
	// 排除的文件模式
	ExcludePatterns []string
	// 进度回调函数
	ProgressCallback func(current, total int64, filename string)
}

// DefaultZipOptions 默认压缩选项
func DefaultZipOptions() ZipOptions {
	return ZipOptions{
		CompressionLevel: zip.Deflate,
		IncludeHidden:    false,
		ExcludePatterns:  []string{},
		ProgressCallback: nil,
	}
}

// ZipDirectory 压缩目录到 ZIP 文件
func ZipDirectory(sourceDir string, zipPath string) error {
	return ZipDirectoryWithOptions(sourceDir, zipPath, DefaultZipOptions())
}

// ZipDirectoryWithOptions 使用自定义选项压缩目录
func ZipDirectoryWithOptions(sourceDir string, zipPath string, options ZipOptions) error {
	// 验证输入参数
	if sourceDir == "" || zipPath == "" {
		return fmt.Errorf("源目录和目标文件路径不能为空")
	}

	// 检查源目录是否存在
	sourceStat, err := os.Stat(sourceDir)
	if err != nil {
		return fmt.Errorf("无法访问源目录 %s: %w", sourceDir, err)
	}
	if !sourceStat.IsDir() {
		return fmt.Errorf("源路径 %s 不是一个目录", sourceDir)
	}

	// 清空目标目录下的所有文件
	if err := cleanBackupDir("backup"); err != nil {
		return err // 或者带提示信息
	}

	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(zipPath), 0755); err != nil {
		return fmt.Errorf("无法创建目标目录: %w", err)
	}

	// 创建 ZIP 文件
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("无法创建 ZIP 文件 %s: %w", zipPath, err)
	}
	defer func() {
		if closeErr := zipFile.Close(); closeErr != nil {
			// 记录关闭错误，但不覆盖主要错误
			fmt.Printf("警告: 关闭 ZIP 文件时出错: %v\n", closeErr)
		}
	}()

	zipWriter := zip.NewWriter(zipFile)
	defer func() {
		if closeErr := zipWriter.Close(); closeErr != nil {
			fmt.Printf("警告: 关闭 ZIP 写入器时出错: %v\n", closeErr)
		}
	}()

	// 计算总文件数量用于进度显示
	var totalFiles int64
	if options.ProgressCallback != nil {
		err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // 跳过错误文件
			}
			if !info.IsDir() && shouldIncludeFile(info, options) {
				totalFiles++
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("计算文件数量时出错: %w", err)
		}
	}

	var processedFiles int64
	sourceDir = filepath.Clean(sourceDir)

	// 遍历目录中的所有文件和子目录
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("遍历文件 %s 时出错: %w", path, err)
		}

		// 检查是否应该包含此文件
		if !shouldIncludeFile(info, options) {
			return nil
		}

		// 构建在 zip 文件中的相对路径
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("计算相对路径失败: %w", err)
		}

		// 标准化路径分隔符为正斜杠（ZIP 标准）
		relPath = filepath.ToSlash(relPath)

		if info.IsDir() {
			// 为目录创建条目
			if relPath != "." {
				_, err := zipWriter.Create(relPath + "/")
				if err != nil {
					return fmt.Errorf("创建目录条目 %s 失败: %w", relPath, err)
				}
			}
			return nil
		}

		// 创建文件条目
		header := &zip.FileHeader{
			Name:     relPath,
			Method:   uint16(options.CompressionLevel),
			Modified: info.ModTime(),
		}

		// 设置文件权限
		header.SetMode(info.Mode())

		zipEntry, err := zipWriter.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("创建 ZIP 条目 %s 失败: %w", relPath, err)
		}

		// 打开原始文件
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("打开文件 %s 失败: %w", path, err)
		}
		defer func() {
			if closeErr := file.Close(); closeErr != nil {
				fmt.Printf("警告: 关闭文件 %s 时出错: %v\n", path, closeErr)
			}
		}()

		// 拷贝文件内容到 zip 条目中
		_, err = io.Copy(zipEntry, file)
		if err != nil {
			return fmt.Errorf("复制文件内容 %s 失败: %w", path, err)
		}

		// 更新进度
		if options.ProgressCallback != nil {
			processedFiles++
			options.ProgressCallback(processedFiles, totalFiles, relPath)
		}

		return nil
	})
}

// shouldIncludeFile 判断是否应该包含文件
func shouldIncludeFile(info os.FileInfo, options ZipOptions) bool {
	filename := info.Name()

	// 检查隐藏文件
	if !options.IncludeHidden && strings.HasPrefix(filename, ".") {
		return false
	}

	// 检查排除模式
	for _, pattern := range options.ExcludePatterns {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return false
		}
	}

	return true
}

// ZipFiles 压缩指定的文件列表
func ZipFiles(files []string, zipPath string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("无法创建 ZIP 文件: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, file := range files {
		err := addFileToZip(zipWriter, file, filepath.Base(file))
		if err != nil {
			return fmt.Errorf("添加文件 %s 到 ZIP 失败: %w", file, err)
		}
	}

	return nil
}

// addFileToZip 将单个文件添加到 ZIP
func addFileToZip(zipWriter *zip.Writer, filename, archiveName string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header := &zip.FileHeader{
		Name:     filepath.ToSlash(archiveName),
		Method:   zip.Deflate,
		Modified: info.ModTime(),
	}
	header.SetMode(info.Mode())

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}

// UnzipFile 解压 ZIP 文件到指定目录
func UnzipFile(src, dest string) error {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("打开 ZIP 文件失败: %w", err)
	}
	defer reader.Close()

	// 确保目标目录存在
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	for _, file := range reader.File {
		err := extractFile(file, dest)
		if err != nil {
			return fmt.Errorf("解压文件 %s 失败: %w", file.Name, err)
		}
	}

	return nil
}

// extractFile 解压单个文件
func extractFile(file *zip.File, destDir string) error {
	filePath := filepath.Join(destDir, file.Name)

	// 防止路径穿越攻击
	if !strings.HasPrefix(filePath, filepath.Clean(destDir)+string(os.PathSeparator)) {
		return fmt.Errorf("无效的文件路径: %s", file.Name)
	}

	if file.FileInfo().IsDir() {
		return os.MkdirAll(filePath, file.FileInfo().Mode())
	}

	// 确保父目录存在
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	fileReader, err := file.Open()
	if err != nil {
		return err
	}
	defer fileReader.Close()

	targetFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.FileInfo().Mode())
	if err != nil {
		return err
	}
	defer targetFile.Close()

	_, err = io.Copy(targetFile, fileReader)
	return err
}

// FileExists 检查文件或目录是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

// cleanBackupDir 清理备份目录
func cleanBackupDir(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("读取备份目录失败: %w", err)
	}

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())
		if err := os.RemoveAll(fullPath); err != nil {
			return fmt.Errorf("删除旧备份失败: %w", err)
		}
	}

	return nil
}
