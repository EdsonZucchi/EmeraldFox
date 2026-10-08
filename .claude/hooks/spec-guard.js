#!/usr/bin/env node
// Hook PreToolUse que restringe onde um agente pode escrever.
//
// Uso:
//   node spec-guard.js only-spec   -> permite escrita apenas dentro de spec/
//   node spec-guard.js no-spec     -> bloqueia qualquer escrita dentro de spec/
//
// Recebe o JSON do hook pela entrada padrão. Sai com código 2 (bloqueio) e uma
// mensagem em stderr quando a escrita viola a regra do agente.

const path = require("path");

const mode = process.argv[2];

let raw = "";
process.stdin.setEncoding("utf8");
process.stdin.on("data", (chunk) => (raw += chunk));
process.stdin.on("end", () => {
  let input;
  try {
    input = JSON.parse(raw);
  } catch {
    process.exit(0);
  }

  const toolInput = input.tool_input || {};
  const target = toolInput.file_path || toolInput.notebook_path;
  if (!target) {
    process.exit(0);
  }

  const projectDir = process.env.CLAUDE_PROJECT_DIR || input.cwd || process.cwd();
  const specDir = path.resolve(projectDir, "spec");
  const resolved = path.resolve(input.cwd || projectDir, target);

  const relative = path.relative(specDir, resolved);
  const insideSpec =
    relative === "" || (!relative.startsWith("..") && !path.isAbsolute(relative));

  if (mode === "only-spec" && !insideSpec) {
    console.error(
      `Bloqueado: o agente arquiteto só pode escrever dentro de spec/. ` +
        `Tentativa de escrita em: ${resolved}`
    );
    process.exit(2);
  }

  if (mode === "no-spec" && insideSpec) {
    console.error(
      `Bloqueado: o agente desenvolvedor não altera a especificação. ` +
        `Registre a divergência no relatório final para o arquiteto. ` +
        `Tentativa de escrita em: ${resolved}`
    );
    process.exit(2);
  }

  process.exit(0);
});
