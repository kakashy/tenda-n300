#!/bin/sh
set -e

BINARY="tenda-n300"
PREFIX="${PREFIX:-/usr/local}"

if [ ! -w "$PREFIX/bin" ]; then
	echo "installing to ~/.local/bin instead of $PREFIX/bin (no write permission)"
	PREFIX="$HOME/.local"
	mkdir -p "$PREFIX/bin"
fi

echo "building $BINARY..."
go build -ldflags="-s -w" -o "$BINARY" .

echo "installing to $PREFIX/bin/$BINARY"
mv "$BINARY" "$PREFIX/bin/$BINARY"

add_line_to_file() {
	_file="$1"
	_line="$2"
	if [ -f "$_file" ] && grep -qsF "$_line" "$_file" 2>/dev/null; then
		return 1
	fi
	echo "$_line" >> "$_file"
	return 0
}

# Install bash completion
if command -v bash >/dev/null 2>&1; then
	BASH_COMPLETION_DIR=""
	for d in /etc/bash_completion.d /usr/local/share/bash-completion/completions; do
		if [ -d "$d" ] && [ -w "$d" ]; then
			BASH_COMPLETION_DIR="$d"
			break
		fi
	done
	if [ -z "$BASH_COMPLETION_DIR" ]; then
		BASH_COMPLETION_DIR="$HOME/.local/share/bash-completion/completions"
		mkdir -p "$BASH_COMPLETION_DIR"
	fi
	"$PREFIX/bin/$BINARY" completion bash > "$BASH_COMPLETION_DIR/$BINARY" 2>/dev/null && echo "bash completion: $BASH_COMPLETION_DIR/$BINARY"
fi

# Install zsh completion
if command -v zsh >/dev/null 2>&1; then
	# Prefer writing to a system fpath directory
	ZSH_COMPLETION_DIR=""
	for d in /usr/local/share/zsh/site-functions /usr/share/zsh/site-functions; do
		if [ -d "$d" ] && [ -w "$d" ]; then
			ZSH_COMPLETION_DIR="$d"
			break
		fi
	done

	if [ -n "$ZSH_COMPLETION_DIR" ]; then
		"$PREFIX/bin/$BINARY" completion zsh > "$ZSH_COMPLETION_DIR/_$BINARY" 2>/dev/null && echo "zsh completion: $ZSH_COMPLETION_DIR/_$BINARY"
	else
		# No system fpath dir — source completion from .zshrc instead
		RC_LINE="source <($PREFIX/bin/$BINARY completion zsh)"
		if add_line_to_file "$HOME/.zshrc" "$RC_LINE"; then
			echo "zsh completion: added source line to ~/.zshrc"
		else
			echo "zsh completion: already configured in ~/.zshrc"
		fi
	fi
fi

echo "done. run \`$BINARY --help\` to get started"
