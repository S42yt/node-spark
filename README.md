## Usage

After compiling this project, you can use it with the following commands:

```bash
# Install a specific version
node-spark install 16.14.0

# Install latest version
node-spark install latest

# Install LTS version
node-spark install lts

# Switch to a different version
node-spark use 16.14.0

# List installed versions
node-spark list

# List available versions
node-spark list --remote

# Remove a version
node-spark remove 16.14.0
```

This Node.js version manager written in Rust is designed to be fast and efficient. It provides essential functionality like installing, switching between versions, listing, and removing Node.js versions.
