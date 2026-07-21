#!/bin/sh
set -e

BINARY="tenda-n300"
PREFIX="${PREFIX:-/usr/local}"

if [ ! -w "$PREFIX/bin" ]; then
	PREFIX="$HOME/.local"
fi

echo "removing $PREFIX/bin/$BINARY"
rm -f "$PREFIX/bin/$BINARY"

# Remove bash completion
for d in /etc/bash_completion.d /usr/local/share/bash-completion/completions "$HOME/.local/share/bash-completion/completions"; do
	if [ -f "$d/$BINARY" ]; then
		echo "removing bash completion: $d/$BINARY"
		rm -f "$d/$BINARY"
	fi
done

# Remove zsh completion
for d in /usr/local/share/zsh/site-functions /usr/share/zsh/site-functions "$HOME/.local/share/zsh/site-functions"; do
	if [ -f "$d/_$BINARY" ]; then
		echo "removing zsh completion: $d/_$BINARY"
		rm -f "$d/_$BINARY"
	fi
done

# Remove zsh completion source line from .zshrc
if [ -f "$HOME/.zshrc" ]; then
	_ZSH_LINE="source <($BINARY completion zsh)"
	grep -sF "$_ZSH_LINE" "$HOME/.zshrc" 2>/dev/null && {
		echo "removing zsh completion line from ~/.zshrc"
		grep -vF "$_ZSH_LINE" "$HOME/.zshrc" > "$HOME/.zshrc.tmp"
		mv "$HOME/.zshrc.tmp" "$HOME/.zshrc"
	}
fi

# Remove config and credentials
CONFIG_DIR="$HOME/.config/tenda-n300"
if [ -d "$CONFIG_DIR" ]; then
	echo "removing config: $CONFIG_DIR"
	rm -rf "$CONFIG_DIR"
fi

echo "done. $BINARY has been uninstalled."
