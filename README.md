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
;; eglot--request は jsonrpc-request を使っている
;; 同期リクエストの場合、jsonrpc のレスポンスの順序が入れ替わると以下のログが出て処理が中断されてしまう
;; [jsonrpc] i[23:14:47.654] anxious continuation to 52 can't run, held up by (51)
;; [jsonrpc] i[23:14:47.808] anxious continuation to 52 running now
;; vue-language-server v3 を使う場合、textDocument/completion の返事が返るまでの間に typescript.tsserverRequest を呼び出す必要があるので、順序が入れ替わる
;; それを回避するために、jsonrpc-async-request を使うように around advice を設定する
(cl-defun my/eglot--request-textdocument-completion-async-advice
    (fn server method params
        &key immediate timeout cancel-on-input cancel-on-input-retval)
  (if (not (eq method :textDocument/completion))
      (funcall fn server method params
               :immediate immediate
               :timeout timeout
               :cancel-on-input cancel-on-input
               :cancel-on-input-retval cancel-on-input-retval)
    (unless immediate (eglot--signal-textDocument/didChange))
    (let (resp err has-input)
      (jsonrpc-async-request server method params
                             :success-fn (lambda (x) (setq resp x))
                             :error-fn (lambda (e) (setq err e))
                             :timeout-fn (lambda () (setq err 'timed-out))
                             :timeout timeout)
      (while (and (null resp) (null err)
                  (and cancel-on-input (not has-input)))
        (unless (sit-for 0.1)
          (setq has-input t)))
      (cond
       ((and cancel-on-input has-input) cancel-on-input-retval)
       (err (error "Completion failure: %s" err))
       (t resp)))))

(advice-add 'eglot--request :around #'my/eglot--request-textdocument-completion-async-advice)

(cl-defmethod eglot-handle-notification
  (server (_method (eql tsserver/request)) params &rest _)
  "Handle notification tsserver/request."
  (pcase-let ((`[,id ,command ,args] params))
    (jsonrpc-async-request
     server
     :workspace/executeCommand `(:command "typescript.tsserverRequest" :arguments [,command ,args])
     :timeout 10
     :success-fn (lambda (resp)
                   (message "typescript.tsserverRequest resp: %s" resp)
                   (jsonrpc-notify server :tsserver/response `[[,id ,(plist-get resp :body)]]))
     :error-fn (lambda (&rest rest)
                 (message "error: failed to sent typescript.tsserverRequest: %s" rest))
     :timeout-fn (lambda ()
                   (message "error: failed to sent typescript.tsserverRequest: timed-out")))))

(add-to-list 'eglot-server-programs
             '(((js-mode :language-id "javascript") (js-ts-mode :language-id "javascript")
                typescript-ts-mode typescript-mode
                vue-mode vue-ts-mode)
               "lspmux" "--config" "/path/to/config.yaml" "--servers" "vuels,tsls"))
```
