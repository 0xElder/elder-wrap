# Elder-Wrap

**Elder-Wrap** is an adapter that enables developers to interact seamlessly with any EVM RollApp connected to the **Elder Chain**. By starting the `elder-wrap` binary, developers can:

1. **Deploy Contracts**: Treat `elder-wrap` as a local EVM node to deploy smart contracts on any RollApp.
2. **Send and Query Transactions**: Use it to send transactions and query their status easily.
3. **Integrate with EVM Tooling**: Connect `elder-wrap` to popular EVM-compatible tools like Hardhat, Truffle, or Remix for a streamlined development experience.

## Key Features
- Acts as a bridge between your local environment and RollApps on the Elder Chain.
- Provides a familiar interface for developers already using EVM tools.
- Simplifies contract deployment and transaction management across RollApps.

## Usage
1. Start the `elder-wrap` binary in your environment.
2. Configure your EVM tooling to point to the `elder-wrap` endpoint (e.g., `http://localhost:8546`).
3. Use your preferred EVM tools as you would with any local node.
4. To know the elder address corresponding to private key visit `http://localhost:8546/elder-address`.

# Please follow the following steps to run Elder-Wrap
```
cp .envrc.example .envrc
# fill appropriate values in .envrc

direnv allow
go build
./main
```
If nothing gets printed after `direnv allow`, direnv is probably not set properly, [refer](https://direnv.net/docs/hook.html#zsh)

## Docker Build Options

You can also build and run Elder-Wrap using Docker:

### Using Docker Bake
```bash
# Build with default settings
docker buildx bake --load 

# Build with custom GitHub token
docker buildx bake --load --set *.args.GITHUB_ACCESS_TOKEN=<your_token>

# Build with custom tag
docker buildx bake --load --set *.tags=elder-wrap:<tag>
```
