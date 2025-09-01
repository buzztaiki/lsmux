# lsp multiplexer

## vue-language-server v3 configuration

`config.yaml`:
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

```

`init.el`:
```elisp
(add-to-list 'eglot-server-programs
             '(((js-mode :language-id "javascript") (js-ts-mode :language-id "javascript")
                typescript-ts-mode typescript-mode
                vue-mode vue-ts-mode)
               "lspmux" "--config" "/path/to/config.yaml" "--servers" "vuels,tsls"))
```
