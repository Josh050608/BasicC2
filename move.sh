#!/bin/bash

# 定义目标目录
TARGET_DIR="$HOME/临时"
mkdir -p "$TARGET_DIR"

# 查找所有有用的文件类型
find . -type f \( \
    -name "*.go" -o \
    -name "*.md" -o \
    -name "*.html" -o \
    -name "go.mod" -o \
    -name "go.sum" -o \
    -name "Makefile" -o \
    -name "*.bat" -o \
    -name "*.ps1" \
\) | while read -r file; do
    # 获取文件名（不含路径）
    filename=$(basename "$file")
    
    # 如果目标文件夹已存在同名文件，则重命名（例如 main.go 变成 main_1.go）
    if [ -f "$TARGET_DIR/$filename" ]; then
        base="${filename%.*}"
        ext="${filename##*.}"
        counter=1
        while [ -f "$TARGET_DIR/${base}_${counter}.${ext}" ]; do
            ((counter++))
        done
        dest_name="${base}_${counter}.${ext}"
    else
        dest_name="$filename"
    fi

    # 复制文件到目标目录
    cp "$file" "$TARGET_DIR/$dest_name"
done

echo "提取完成！所有文件已平铺存放在: $TARGET_DIR"
ls "$TARGET_DIR"