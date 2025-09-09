package lsmux

func IsMethodSupported(method string, supportedCaps map[string]struct{}) bool {
	methodCap, useCap := MethodToCapability[method]
	if !useCap {
		return true
	}

	_, supported := supportedCaps[methodCap]
	return supported
}

// CollectSupportedCapabilities returns a map of dot notated capability to whether it's supported or not.
func CollectSupportedCapabilities(kvCaps map[string]any) map[string]struct{} {
	res := map[string]struct{}{}
	collectSupportedCapabilities("", kvCaps, res)
	return res
}

func collectSupportedCapabilities(prefix string, kvCaps map[string]any, res map[string]struct{}) {
	for k, v := range kvCaps {
		switch v := v.(type) {
		case map[string]any:
			res[prefix+k] = struct{}{}
			collectSupportedCapabilities(prefix+k+".", v, res)
		case bool:
			if v {
				res[prefix+k] = struct{}{}
			}
		default:
			if v != nil {
				res[prefix+k] = struct{}{}
			}
		}
	}
}

// TODO go generate?
// curl -sSf https://raw.githubusercontent.com/microsoft/vscode-languageserver-node/refs/heads/main/protocol/metaModel.json | jq '.requests[]|select(.serverCapability)|"\t\(.method|@json): \(.serverCapability|@json),"' -r
var MethodToCapability = map[string]string{
	"textDocument/implementation":            "implementationProvider",
	"textDocument/typeDefinition":            "typeDefinitionProvider",
	"workspace/workspaceFolders":             "workspace.workspaceFolders",
	"textDocument/documentColor":             "colorProvider",
	"textDocument/colorPresentation":         "colorProvider",
	"textDocument/foldingRange":              "foldingRangeProvider",
	"textDocument/declaration":               "declarationProvider",
	"textDocument/selectionRange":            "selectionRangeProvider",
	"textDocument/prepareCallHierarchy":      "callHierarchyProvider",
	"textDocument/semanticTokens/full":       "semanticTokensProvider",
	"textDocument/semanticTokens/full/delta": "semanticTokensProvider.full.delta",
	"textDocument/semanticTokens/range":      "semanticTokensProvider.range",
	"textDocument/linkedEditingRange":        "linkedEditingRangeProvider",
	"workspace/willCreateFiles":              "workspace.fileOperations.willCreate",
	"workspace/willRenameFiles":              "workspace.fileOperations.willRename",
	"workspace/willDeleteFiles":              "workspace.fileOperations.willDelete",
	"textDocument/moniker":                   "monikerProvider",
	"textDocument/prepareTypeHierarchy":      "typeHierarchyProvider",
	"textDocument/inlineValue":               "inlineValueProvider",
	"textDocument/inlayHint":                 "inlayHintProvider",
	"inlayHint/resolve":                      "inlayHintProvider.resolveProvider",
	"textDocument/diagnostic":                "diagnosticProvider",
	"workspace/diagnostic":                   "diagnosticProvider.workspaceDiagnostics",
	"textDocument/inlineCompletion":          "inlineCompletionProvider",
	"workspace/textDocumentContent":          "workspace.textDocumentContent",
	"textDocument/willSaveWaitUntil":         "textDocumentSync.willSaveWaitUntil",
	"textDocument/completion":                "completionProvider",
	"completionItem/resolve":                 "completionProvider.resolveProvider",
	"textDocument/hover":                     "hoverProvider",
	"textDocument/signatureHelp":             "signatureHelpProvider",
	"textDocument/definition":                "definitionProvider",
	"textDocument/references":                "referencesProvider",
	"textDocument/documentHighlight":         "documentHighlightProvider",
	"textDocument/documentSymbol":            "documentSymbolProvider",
	"textDocument/codeAction":                "codeActionProvider",
	"codeAction/resolve":                     "codeActionProvider.resolveProvider",
	"workspace/symbol":                       "workspaceSymbolProvider",
	"workspaceSymbol/resolve":                "workspaceSymbolProvider.resolveProvider",
	"textDocument/codeLens":                  "codeLensProvider",
	"codeLens/resolve":                       "codeLensProvider.resolveProvider",
	"textDocument/documentLink":              "documentLinkProvider",
	"documentLink/resolve":                   "documentLinkProvider.resolveProvider",
	"textDocument/formatting":                "documentFormattingProvider",
	"textDocument/rangeFormatting":           "documentRangeFormattingProvider",
	"textDocument/rangesFormatting":          "documentRangeFormattingProvider.rangesSupport",
	"textDocument/onTypeFormatting":          "documentOnTypeFormattingProvider",
	"textDocument/rename":                    "renameProvider",
	"textDocument/prepareRename":             "renameProvider.prepareProvider",
	"workspace/executeCommand":               "executeCommandProvider",
}
