package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	srcDir       string
	dstDir       string
	syncInterval time.Duration
	ignoreStr    string
	ignoreList   []string
	fileModTimes = make(map[string]time.Time)
)

func main() {
	flag.StringVar(&srcDir, "src", "", "Source directory to sync from")
	flag.StringVar(&dstDir, "dst", "", "Destination directory to sync to")
	flag.DurationVar(&syncInterval, "interval", 2*time.Second, "Sync polling interval")
	flag.StringVar(&ignoreStr, "ignore", ".DS_Store,desktop.ini,Thumbs.db,Zone.Identifier,.obsidian/workspace,.trash", "Comma-separated list of file/folder names to ignore")
	flag.Parse()

	if srcDir == "" || dstDir == "" {
		log.Fatal("Error: --src and --dst flags are required.\nUsage: dirmirror --src <dir1> --dst <dir2>")
	}

	if ignoreStr != "" {
		ignoreList = strings.Split(ignoreStr, ",")
		for i := range ignoreList {
			ignoreList[i] = strings.TrimSpace(ignoreList[i])
		}
	}

	// 【安全装置】両方のディレクトリが存在するか確認
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		log.Fatalf("ソースディレクトリが見つかりません: %v", err)
	}
	if _, err := os.Stat(dstDir); os.IsNotExist(err) {
		log.Fatalf("ターゲットディレクトリが見つかりません: %v", err)
	}

	fmt.Println("🚀 起動時の差分同期（マージ）を実行中...")
	initialSync(srcDir, dstDir)

	fmt.Printf("✅ 監視を開始しました (インターバル: %v)\n", syncInterval)
	fmt.Printf("監視中 (Src): %s\n", srcDir)
	fmt.Printf("監視中 (Dst): %s\n", dstDir)

	for {
		time.Sleep(syncInterval)
		checkForChanges(srcDir, dstDir, "Src -> Dst")
		checkForChanges(dstDir, srcDir, "Dst -> Src")
	}
}

// ---------------------------------------------------------
// 1. 無視ファイルの判定ロジック
// ---------------------------------------------------------
func shouldIgnore(path string) bool {
	slashPath := filepath.ToSlash(path)
	base := filepath.Base(slashPath)

	for _, ignored := range ignoreList {
		if ignored == "" {
			continue
		}
		ignored = filepath.ToSlash(ignored)
		
		// 拡張子/ファイル名の完全一致、またはパスの一部として含まれるか
		if strings.HasSuffix(base, ignored) || 
		   slashPath == ignored || 
		   strings.HasPrefix(slashPath, ignored+"/") || 
		   strings.Contains(slashPath, "/"+ignored+"/") || 
		   strings.HasSuffix(slashPath, "/"+ignored) {
			return true
		}
	}

	return false
}

// ---------------------------------------------------------
// 2. 起動時の初期同期（マージ）ロジック
// ---------------------------------------------------------
func initialSync(dirA, dirB string) {
	filesA := getFilesInfo(dirA)
	filesB := getFilesInfo(dirB)

	// Aを基準にBと比較
	for relPath, infoA := range filesA {
		pathA := filepath.Join(dirA, relPath)
		pathB := filepath.Join(dirB, relPath)
		infoB, existsInB := filesB[relPath]

		if !existsInB {
			// Bに無い -> AからBへコピー（オフライン中の新規作成とみなす）
			copyFile(pathA, pathB)
			log.Printf("🔄 初期同期 (復元): %s -> Dst\n", relPath)
		} else if infoA.ModTime().Sub(infoB.ModTime()) > time.Second {
			// Aの方が新しい -> AからBへ上書きコピー
			copyFile(pathA, pathB)
			log.Printf("🔄 初期同期 (更新): %s -> Dst\n", relPath)
		}
	}

	// Bを基準にAと比較（Aに無いものを探す）
	for relPath, infoB := range filesB {
		pathA := filepath.Join(dirA, relPath)
		pathB := filepath.Join(dirB, relPath)
		infoA, existsInA := filesA[relPath]

		if !existsInA {
			// Aに無い -> BからAへコピー
			copyFile(pathB, pathA)
			log.Printf("🔄 初期同期 (復元): Dst -> %s\n", relPath)
		} else if infoB.ModTime().Sub(infoA.ModTime()) > time.Second {
			// Bの方が新しい -> BからAへ上書きコピー
			copyFile(pathB, pathA)
			log.Printf("🔄 初期同期 (更新): Dst -> %s\n", relPath)
		}
	}

	// マージ完了後の最新状態をマップに記録
	recordAllFileTimes(dirA)
	recordAllFileTimes(dirB)
}

// 指定ディレクトリの全ファイル情報を取得（相対パスキーのマップを返す）
func getFilesInfo(baseDir string) map[string]os.FileInfo {
	result := make(map[string]os.FileInfo)
	filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || shouldIgnore(path) {
			return nil
		}
		if info, err := d.Info(); err == nil {
			rel, _ := filepath.Rel(baseDir, path)
			result[rel] = info
		}
		return nil
	})
	return result
}

// 状態記録
func recordAllFileTimes(baseDir string) {
	filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || shouldIgnore(path) {
			return nil
		}
		if info, err := d.Info(); err == nil {
			fileModTimes[path] = info.ModTime()
		}
		return nil
	})
}

// ---------------------------------------------------------
// 3. 常駐監視・同期ロジック（ポーリング）
// ---------------------------------------------------------
func checkForChanges(srcDir, dstDir, direction string) {
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return
	}

	currentFiles := make(map[string]bool)

	// 更新・追加の検知
	filepath.WalkDir(srcDir, func(srcPath string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || shouldIgnore(srcPath) {
			return nil
		}

		currentFiles[srcPath] = true
		info, err := d.Info()
		if err != nil {
			return nil
		}

		lastTime, exists := fileModTimes[srcPath]

		// 新規 or 更新
		if !exists || info.ModTime().Sub(lastTime) > time.Second {
			relPath, _ := filepath.Rel(srcDir, srcPath)
			dstPath := filepath.Join(dstDir, relPath)

			if err := copyFile(srcPath, dstPath); err == nil {
				log.Printf("✅ コピー [%s]: %s\n", direction, relPath)
				fileModTimes[srcPath] = info.ModTime()
				if dstInfo, err := os.Stat(dstPath); err == nil {
					fileModTimes[dstPath] = dstInfo.ModTime()
				}
			}
		}
		return nil
	})

	// 削除検知
	for trackedPath := range fileModTimes {
		if strings.HasPrefix(trackedPath, srcDir) && !currentFiles[trackedPath] && !shouldIgnore(trackedPath) {
			relPath, _ := filepath.Rel(srcDir, trackedPath)
			dstPath := filepath.Join(dstDir, relPath)

			err := os.Remove(dstPath)
			if err == nil || os.IsNotExist(err) {
				log.Printf("🗑️ 削除 [%s]: %s\n", direction, relPath)
				delete(fileModTimes, trackedPath)
				delete(fileModTimes, dstPath)
			}
		}
	}
}

// ファイルコピー（変更なし）
func copyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("not a regular file")
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	destination.Close() // Chtimesの前に確実にクローズする

	// ファイルの更新日時（ModTime）をコピー元に合わせる
	if err == nil {
		if errTime := os.Chtimes(dst, sourceFileStat.ModTime(), sourceFileStat.ModTime()); errTime != nil {
			log.Printf("⚠️ 更新日時の同期に失敗 [%s]: %v\n", dst, errTime)
		}
	}

	return err
}
