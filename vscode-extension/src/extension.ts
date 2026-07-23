import * as vscode from 'vscode';
import { exec } from 'child_process';
import * as path from 'path';
import * as fs from 'fs';

let diagnosticCollection: vscode.DiagnosticCollection;

interface RazifyCheckOutput {
    env_file: string;
    example_file: string;
    status: string;
    validation_results: {
        key: string;
        status: string;
        message: string;
    }[];
    secret_scan_results: {
        line: number;
        key: string;
        value: string;
        reason: string;
        risk: string;
    }[];
}

export function activate(context: vscode.ExtensionContext) {
    diagnosticCollection = vscode.languages.createDiagnosticCollection('razify');
    context.subscriptions.push(diagnosticCollection);

    // Run diagnostics on open and save
    if (vscode.window.activeTextEditor) {
        runDiagnostics(vscode.window.activeTextEditor.document);
    }

    context.subscriptions.push(
        vscode.workspace.onDidSaveTextDocument((document) => {
            const config = vscode.workspace.getConfiguration('razify');
            if (config.get<boolean>('checkOnSave', true)) {
                runDiagnostics(document);
            }
        })
    );

    context.subscriptions.push(
        vscode.workspace.onDidOpenTextDocument((document) => {
            runDiagnostics(document);
        })
    );

    // Register Hover Provider for .env files
    context.subscriptions.push(
        vscode.languages.registerHoverProvider(
            [{ language: 'properties' }, { language: 'dotenv' }, { pattern: '**/.env*' }],
            new RazifyHoverProvider()
        )
    );

    // Register Auto-Completion Provider
    context.subscriptions.push(
        vscode.languages.registerCompletionItemProvider(
            [{ language: 'properties' }, { language: 'dotenv' }, { pattern: '**/.env*' }],
            new RazifyCompletionProvider()
        )
    );

    // Register Commands
    context.subscriptions.push(
        vscode.commands.registerCommand('razify.check', () => {
            if (vscode.window.activeTextEditor) {
                runDiagnostics(vscode.window.activeTextEditor.document, true);
            } else {
                vscode.window.showInformationMessage('Razify: Open a .env file to run integrity check.');
            }
        }),

        vscode.commands.registerCommand('razify.fix', () => {
            runCliCommand('fix', 'Successfully synced missing environment keys.');
        }),

        vscode.commands.registerCommand('razify.init', () => {
            runCliCommand('init', 'Initialized .env.example template.');
        }),

        vscode.commands.registerCommand('razify.guardInstall', () => {
            runCliCommand('guard install', 'Installed Razify git pre-commit hook.');
        })
    );
}

function runDiagnostics(document: vscode.TextDocument, showStatusMessage = false) {
    const fileName = path.basename(document.fileName);
    if (!fileName.startsWith('.env')) {
        return;
    }

    const config = vscode.workspace.getConfiguration('razify');
    if (!config.get<boolean>('enable', true)) {
        diagnosticCollection.clear();
        return;
    }

    const workspaceFolder = vscode.workspace.getWorkspaceFolder(document.uri);
    const cwd = workspaceFolder ? workspaceFolder.uri.fsPath : path.dirname(document.fileName);
    const execPath = config.get<string>('executablePath', 'razify');

    const cmd = `${execPath} check --json`;

    exec(cmd, { cwd }, (error, stdout) => {
        if (!stdout) {
            return;
        }

        try {
            const output: RazifyCheckOutput = JSON.parse(stdout);
            const diagnostics: vscode.Diagnostic[] = [];

            // 1. Process Validation Errors
            for (const val of output.validation_results || []) {
                if (val.status === 'MISSING' || val.status === 'INVALID') {
                    const lineIndex = findKeyLine(document, val.key);
                    const range = new vscode.Range(
                        new vscode.Position(lineIndex, 0),
                        new vscode.Position(lineIndex, 200)
                    );
                    const diag = new vscode.Diagnostic(
                        range,
                        `Razify Validation [${val.status}]: ${val.key} — ${val.message}`,
                        vscode.DiagnosticSeverity.Error
                    );
                    diag.code = 'razify-validation';
                    diagnostics.push(diag);
                } else if (val.status === 'EMPTY' || val.status === 'PLACEHOLDER') {
                    const lineIndex = findKeyLine(document, val.key);
                    const range = new vscode.Range(
                        new vscode.Position(lineIndex, 0),
                        new vscode.Position(lineIndex, 200)
                    );
                    const diag = new vscode.Diagnostic(
                        range,
                        `Razify Warning [${val.status}]: ${val.key} — ${val.message}`,
                        vscode.DiagnosticSeverity.Warning
                    );
                    diag.code = 'razify-warning';
                    diagnostics.push(diag);
                }
            }

            // 2. Process Secret Leaks
            for (const sec of output.secret_scan_results || []) {
                const lineIndex = sec.line > 0 ? sec.line - 1 : findKeyLine(document, sec.key);
                const range = new vscode.Range(
                    new vscode.Position(lineIndex, 0),
                    new vscode.Position(lineIndex, 200)
                );
                const severity = sec.risk === 'CRITICAL' || sec.risk === 'HIGH' 
                    ? vscode.DiagnosticSeverity.Error 
                    : vscode.DiagnosticSeverity.Warning;

                const diag = new vscode.Diagnostic(
                    range,
                    `Razify Security [${sec.risk}]: Leaked credential detected (${sec.reason})`,
                    severity
                );
                diag.code = 'razify-secret-leak';
                diagnostics.push(diag);
            }

            diagnosticCollection.set(document.uri, diagnostics);

            if (showStatusMessage) {
                if (output.status === 'PASSED') {
                    vscode.window.showInformationMessage('⚡ Razify: Environment configuration is secure & valid!');
                } else {
                    vscode.window.showWarningMessage('⚡ Razify: Action required on environment configuration.');
                }
            }
        } catch {
            // Ignore JSON parse errors
        }
    });
}

function findKeyLine(document: vscode.TextDocument, key: string): number {
    for (let i = 0; i < document.lineCount; i++) {
        const line = document.lineAt(i).text;
        if (line.trim().startsWith(`${key}=`)) {
            return i;
        }
    }
    return 0;
}

class RazifyHoverProvider implements vscode.HoverProvider {
    provideHover(document: vscode.TextDocument, position: vscode.Position): vscode.Hover | undefined {
        const lineText = document.lineAt(position.line).text;
        if (!lineText.includes('=')) {
            return undefined;
        }

        const key = lineText.split('=')[0].trim();
        const workspaceFolder = vscode.workspace.getWorkspaceFolder(document.uri);
        const cwd = workspaceFolder ? workspaceFolder.uri.fsPath : path.dirname(document.fileName);
        const examplePath = path.join(cwd, '.env.example');

        if (!fs.existsSync(examplePath)) {
            return undefined;
        }

        const exampleContent = fs.readFileSync(examplePath, 'utf-8');
        const lines = exampleContent.split('\n');

        for (let i = 0; i < lines.length; i++) {
            const l = lines[i].trim();
            if (l.startsWith(`${key}=`)) {
                let comment = '';
                if (i > 0 && lines[i - 1].trim().startsWith('#')) {
                    comment = lines[i - 1].trim().replace(/^#\s*/, '');
                }

                const markdown = new vscode.MarkdownString();
                markdown.appendMarkdown(`### ⚡ Razify Schema Info: \`${key}\`\n\n`);
                if (comment) {
                    markdown.appendMarkdown(`**Schema Rules & Comment:**\n\`${comment}\`\n\n`);
                }
                markdown.appendMarkdown(`*Defined in \`.env.example\`*`);
                return new vscode.Hover(markdown);
            }
        }

        return undefined;
    }
}

class RazifyCompletionProvider implements vscode.CompletionItemProvider {
    provideCompletionItems(document: vscode.TextDocument): vscode.CompletionItem[] {
        const workspaceFolder = vscode.workspace.getWorkspaceFolder(document.uri);
        const cwd = workspaceFolder ? workspaceFolder.uri.fsPath : path.dirname(document.fileName);
        const examplePath = path.join(cwd, '.env.example');

        if (!fs.existsSync(examplePath)) {
            return [];
        }

        const items: vscode.CompletionItem[] = [];
        const exampleContent = fs.readFileSync(examplePath, 'utf-8');
        const lines = exampleContent.split('\n');

        for (const line of lines) {
            const l = line.trim();
            if (l && !l.startsWith('#') && l.includes('=')) {
                const key = l.split('=')[0].trim();
                const item = new vscode.CompletionItem(key, vscode.CompletionItemKind.Variable);
                item.detail = 'Razify Environment Key';
                item.documentation = new vscode.MarkdownString(`Imported from \`.env.example\``);
                items.push(item);
            }
        }

        return items;
    }
}

function runCliCommand(subcmd: string, successMessage: string) {
    const config = vscode.workspace.getConfiguration('razify');
    const execPath = config.get<string>('executablePath', 'razify');
    const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
    const cwd = workspaceFolder ? workspaceFolder.uri.fsPath : process.cwd();

    exec(`${execPath} ${subcmd}`, { cwd }, (error, stdout) => {
        if (error) {
            vscode.window.showErrorMessage(`Razify Error: ${error.message}`);
        } else {
            vscode.window.showInformationMessage(`⚡ Razify: ${successMessage}`);
        }
    });
}

export function deactivate() {
    if (diagnosticCollection) {
        diagnosticCollection.clear();
        diagnosticCollection.dispose();
    }
}
