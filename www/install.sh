#!/usr/bin/env sh

releases_url="https://api.github.com/repos/eighty4/maestro/releases/latest"
releases_response=""
if which curl >/dev/null 2>&1; then
  releases_response=$(curl -s "$releases_url")
elif which wget >/dev/null 2>&1; then
  releases_response=$(wget -q -O - -o /dev/null "$url")
else
  use_go_install "unable to download binary"
fi
latest_version=$(echo "$releases_response" | grep \"tag_name\": | cut -d : -f 2 | cut -d \" -f 2)

echo "installing maestro@$latest_version"
echo ""

cpu_architecture=""
case $(uname -m) in
  i386)              use_go_install "no prebuilt binary for i386" ;;
  i686)              use_go_install "no prebuilt binary for i686" ;;
  x86_64)            cpu_architecture="amd64" ;;
  arm)               use_go_install "no prebuilt binary for 32-bit arm" ;; # todo check if 64 bit arm with dpkg?
  aarch64 | arm64)   cpu_architecture="arm64" ;;
  *)                 use_go_install "unable to figure out cpu architecture" ;;
esac

os_platform=""
case $(uname -o) in
  Darwin)            os_platform="darwin" ;;
  GNU/Linux)         os_platform="linux" ;;
  *)                 use_go_install "unable to figure out operating system" ;;
esac

shell_profile=""
if [ -z "$BASH_VERSION" ]; then
  shell_profile=".profile"
elif [ -z "$ZSH_VERSION" ]; then
  shell_profile=".zprofile"
else
  use_go_install "shell $SHELL is not supported. visit https://github.com/eighty4/maestro/issues to submit a PR."
fi

# download binary
install_dir="$HOME/.maestro/bin"
mkdir -p "$install_dir"
url="https://github.com/eighty4/maestro/releases/download/$latest_version/maestro-$os_platform-$cpu_architecture"
if which curl >/dev/null 2>&1; then
  curl -Ls "$url" -o "$install_dir/maestro"
elif which wget >/dev/null 2>&1; then
  wget -q -O "$install_dir/maestro" -o /dev/null "$url"
else
  use_go_install "unable to download binary"
fi
chmod +x "$install_dir/maestro"

if ! grep .maestro/bin "$HOME/$shell_profile" >/dev/null 2>&1; then
  { echo ""; echo "# added by https://raw.githubusercontent.com/eighty4/maestro/main/install.sh"; } >> "$HOME/$shell_profile"
  echo "PATH=\"\$PATH:$install_dir"\" >> "$HOME/$shell_profile"
fi

checkmark="\033[1;38;5;41m\0342\0234\0224\033[m"
echo "$checkmark binary installed at \033[1m~/.maestro/bin/maestro\033[m"
echo "$checkmark \033[1m~/$shell_profile\033[m now adds maestro to PATH"
echo ""
echo "run these commands to verify install:"
echo "  source ~/$shell_profile"
echo "  maestro"

use_go_install() {
  if [ $# -gt 0 ]
  then
    echo "$1"
  fi
  echo "visit https://github.com/eighty4/maestro to install with go"
  exit 1
}
