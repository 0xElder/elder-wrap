# elder-wrap
```
cp .envrc.example .envrc
# fill appropriate values in .envrc

direnv allow
go build
./main
```
if nothing gets printed after `direnv allow`, direnv is probably not set properly, [refer](https://direnv.net/docs/hook.html#zsh)
