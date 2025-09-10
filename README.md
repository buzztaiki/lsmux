# lsmux

A language server multiplexer.

## Installation

```console
% go install github.com/gnolang/lsmux/cmd/lsmux@latest
```

## Usage

Put your configuration to `$HOME/.config/lsmux/config.yaml` as shown below:

```yaml
servers:
  - name: tsls
    command: typescript-language-server
    args: [--stdio]
    initializationOptions:
      plugins:
        - name: "@vue/typescript-plugin"
          location: /usr/lib/node_modules/@vue/language-server
          languages: ["vue"]
          configNamespace: typescript

  - name: vuels
    command: vue-language-server
    args: [--stdio]

  - name: eslint
    command: eslint-language-server
    args: [--stdio]

  - name: pyright
    command: pyright-langserver
    args: [--stdio]

  - name: ruff
    command: ruff
    args: [server]
```


Run it by specifying the language servers you want to run simultaneously with the `--servers` option as shown below:

```console
% lsmux --servers tsls,vuels,eslint
% lsmux --servers pyright,ruff
```

Write it as follows for Eglot:

```elisp
(add-to-list 'eglot-server-programs
             '(((js-mode :language-id "javascript") (js-ts-mode :language-id "javascript")
                typescript-mode typescript-ts-mode
                vue-mode vue-ts-mode)
               "lsmux" "--servers" "tsls,vuels,eslint"))
(add-to-list 'eglot-server-programs
             '((python-ts-mode python-mode) "lsmux" "--servers" "pyright,ruff"))
```

## Features
- Merge completion results from all servers.
- Merge Diagnostics notifications from all servers.
- Dispatch Code Action and Execute Command.
- Transfer requests other than the above to the first capable server.
- Transfer notifications to all servers.
- Support `tsserver/request` for vuels v3.

## Alternatives

- https://github.com/thefrontside/lspx
- https://gitlab.com/tmtms/lsp_router
- https://github.com/garyo/lsp-multiplexer

## License

MIT
