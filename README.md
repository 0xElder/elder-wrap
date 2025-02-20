# Elder-Wrap

**Elder-Wrap** is an adapter that enables developers to interact seamlessly with any EVM RollApp connected to the **Elder Chain**. By starting the `elder-wrap` binary, developers can:

1. **Deploy Contracts**: Treat `elder-wrap` as a local EVM node to deploy smart contracts on any RollApp.
2. **Send and Query Transactions**: Use it to send transactions and query their status easily.
3. **Integrate with EVM Tooling**: Connect `elder-wrap` to popular EVM-compatible tools like Hardhat, Truffle, or Remix for a streamlined development experience.
4. **Manage Keys**: Use the built-in keystore functionality to manage your keys securely.

## Key Features
- Acts as a bridge between your local environment and RollApps on the Elder Chain.
- Provides a familiar interface for developers already using EVM tools.
- Simplifies contract deployment and transaction management across RollApps.

## Usage
1. Use the `elder-wrap` binary to start the server in local or manage keys.
2. Configure your EVM tooling to point to the `elder-wrap` endpoint (e.g., `http://localhost:8546/rollApp_alias`).
3. Use your preferred EVM tools as you would with any local node.
4. To know the elder address corresponding to private key use keystore commands.

# Steps to use elder-wrap

```
cp config.yaml.sample config.yaml
# fill appropriate values in config.yaml

go build -o elder-wrap
```

## To start server
```
./elder-wrap server
```
## To use Keystore
```
./elder-wrap keystore
```

## API Endpoints
Base endpoint: `http://localhost:8546`

#### List RollApp Configurations
- **GET /** 
  - Returns all available RollApp endpoints and their configurations
  - Response example:
    ```json
    {
      "elder_grpc": "localhost:9090",
      "endpoints": {
        "rollapp1": {
          "endpoint": "/rollapp1",
          "rpc": "http://localhost:8545",
          "elder_registration_id": 1
        }
      }
    }
    ```

#### Send Transactions
- **POST /{rollapp-name}**
  - Use this directly in your dApp to send transactions to RollApps
  - Example `ROLL_APP_RPC : base_url/rollapp1`

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
