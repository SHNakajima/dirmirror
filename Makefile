# 変数定義
BINARY_NAME=dirmirror.exe
GOOS=windows
GOARCH=amd64

.PHONY: all build clean run

# デフォルトのターゲット
all: build

# Windows向けクロスコンパイル
build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BINARY_NAME) main.go
	@echo "🚨 Windows用バイナリ $(BINARY_NAME) のビルドが完了しました。"

# 前回のビルド成果物を削除
clean:
	rm -f $(BINARY_NAME)
	@echo "🧹 クリーンアップが完了しました。"

# WSLからWindowsプロセスとしてテスト実行
run: build
	./$(BINARY_NAME)