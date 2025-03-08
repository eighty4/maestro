# Maestro CLI

Stylish developer workflows.

```bash
cargo install maestro_cli
```

## `maestro git [-i, --interactive]`

Sync a workspace of local repositories from their upstream remotes:

```bash
# make a workspace
mkdir workspace && cd workspace
git clone https://github.com/bytecodealliance/wasmtime.git
git clone https://github.com/llvm/llvm-project.git
git clone https://github.com/ziglang/zig.git
git clone https://github.com/microsoft/monaco-editor.git

# drop latest commits from each repo
find . -maxdepth 1 -type d \( ! -name . \) -exec bash -c "cd '{}' && git reset --hard HEAD~2" \;

# pull workspace repos
maestro git --interactive
```

Maestro will only perform ff merges.

The output summarizes each repository's state to quickly check the status of your workspace.
