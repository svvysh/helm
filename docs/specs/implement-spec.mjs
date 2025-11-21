#!/usr/bin/env node
import fs from 'fs/promises';
import path from 'path';
import { spawn } from 'child_process';

async function main() {
  const argv = process.argv.slice(2);
  if (!argv[0]) {
    console.error('Usage: implement-spec.mjs <spec-dir>');
    process.exit(1);
  }

  const specArg = argv[0];
  const cwd = process.cwd();

  let specDirCandidate;
  if (path.isAbsolute(specArg)) {
    specDirCandidate = specArg;
  } else {
    specDirCandidate = path.resolve(cwd, specArg);
  }

  try {
    await fs.access(specDirCandidate);
  } catch {
    const alt = path.resolve(cwd, 'docs', 'specs', specArg);
    try {
      await fs.access(alt);
      specDirCandidate = alt;
    } catch {
      console.error('Could not find spec directory at', specDirCandidate, 'or', alt);
      process.exit(1);
    }
  }

  const specDir = specDirCandidate;
  const specsRoot = path.dirname(specDir);

  const metaPath = path.join(specDir, 'metadata.json');
  const specMdPath = path.join(specDir, 'SPEC.md');
  const checklistPath = path.join(specDir, 'acceptance-checklist.md');
  const implTemplatePath = path.join(specsRoot, 'implement.prompt-template.md');
  const reviewTemplatePath = path.join(specsRoot, 'review.prompt-template.md');
  const settingsPath = path.join(specsRoot, '.cli-settings.json');
  const reportPath = path.join(specDir, 'implementation-report.md');

  let meta;
  try {
    meta = JSON.parse(await fs.readFile(metaPath, 'utf8'));
  } catch (err) {
    console.error('Failed to read metadata.json:', err);
    process.exit(1);
  }

  let specBody;
  try {
    specBody = await fs.readFile(specMdPath, 'utf8');
  } catch (err) {
    console.error('Failed to read SPEC.md:', err);
    process.exit(1);
  }

  let checklist = '';
  try {
    checklist = await fs.readFile(checklistPath, 'utf8');
  } catch {
    // optional
  }

  let implTpl, reviewTpl;
  try {
    implTpl = await fs.readFile(implTemplatePath, 'utf8');
    reviewTpl = await fs.readFile(reviewTemplatePath, 'utf8');
  } catch (err) {
    console.error('Failed to read prompt templates:', err);
    process.exit(1);
  }

  let settings = {};
  try {
    const raw = await fs.readFile(settingsPath, 'utf8');
    settings = JSON.parse(raw);
  } catch {
    settings = {};
  }

  const mode = settings.mode || 'strict';
  const acceptanceCommands = meta.acceptanceCommands || settings.acceptanceCommands || [];
  const acceptanceCommandsText = acceptanceCommands.length
    ? acceptanceCommands.map(c => `- ${c}`).join('\n')
    : '- (none specified)';

  const maxAttempts = parseInt(process.env.MAX_ATTEMPTS || settings.defaultMaxAttempts || '2', 10);
  const modelImpl = process.env.CODEX_MODEL_IMPL || settings.codexModelRunImpl || 'gpt-5.1-codex';
  const modelVer = process.env.CODEX_MODEL_VER || settings.codexModelRunVer || 'gpt-5.1-codex';

  const specID = meta.id || path.basename(specDir);
  const specName = meta.name || extractTitle(specBody) || '(unnamed spec)';

  let remainingTasks = [];

  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    const now = new Date();

    console.log(`\n=== Attempt ${attempt} of ${maxAttempts} for ${specID} ===\n`);

    const workerPrompt = implTpl
      .replace(/{{SPEC_ID}}/g, specID)
      .replace(/{{SPEC_NAME}}/g, specName)
      .replace(/{{SPEC_BODY}}/g, specBody)
      .replace(/{{ACCEPTANCE_COMMANDS}}/g, acceptanceCommandsText)
      .replace(/{{PREVIOUS_REMAINING_TASKS}}/g, JSON.stringify(remainingTasks, null, 2))
      .replace(/{{MODE}}/g, mode);

    const workerOutput = await runCodex(workerPrompt, false, modelImpl);

    const reviewInput = reviewTpl
      .replace(/{{SPEC_ID}}/g, specID)
      .replace(/{{SPEC_NAME}}/g, specName)
      .replace(/{{SPEC_BODY}}/g, specBody)
      .replace(/{{ACCEPTANCE_CHECKLIST}}/g, checklist)
      .replace(/{{ACCEPTANCE_COMMANDS}}/g, acceptanceCommandsText)
      .replace(/{{IMPLEMENTATION_REPORT}}/g, workerOutput)
      .replace(/{{MODE}}/g, mode);

    const verifierOutput = await runCodex(reviewInput, true, modelVer);

    const lines = verifierOutput.split(/\r?\n/).filter(Boolean);
    if (lines.length < 2) {
      console.error('Verifier output did not contain at least two non-empty lines');
      process.exit(1);
    }

    const statusLine = lines[0].trim();
    const jsonLine = lines[1].trim();

    let status;
    if (statusLine === 'STATUS: ok') {
      status = 'ok';
    } else if (statusLine === 'STATUS: missing') {
      status = 'missing';
    } else {
      console.error('Unexpected status line from verifier:', statusLine);
      process.exit(1);
    }

    let json;
    try {
      json = JSON.parse(jsonLine);
    } catch (err) {
      console.error('Failed to parse verifier JSON line:', err);
      process.exit(1);
    }
    remainingTasks = Array.isArray(json.remainingTasks) ? json.remainingTasks : [];

    const notePrefix = `[${now.toISOString()}] attempt ${attempt} status=${status}`;
    if (status === 'ok') {
      meta.status = 'done';
      meta.lastRun = now.toISOString();
      meta.notes = (meta.notes || '') + `\n${notePrefix}: ok`;
    } else {
      meta.status = 'in-progress';
      meta.lastRun = now.toISOString();
      meta.notes = (meta.notes || '') + `\n${notePrefix}: remaining tasks: ${remainingTasks.join('; ')}`;
    }

    await fs.writeFile(metaPath, JSON.stringify(meta, null, 2), 'utf8');

    const report = [
      `# Implementation Report for ${specID} — ${specName}`.trim(),
      '',
      `- Mode: ${mode}`,
      `- Max attempts: ${maxAttempts}`,
      `- Attempts used: ${attempt}`,
      `- Final verifier status: ${status}`,
      '',
      '## Remaining tasks',
      '',
      JSON.stringify({ remainingTasks }, null, 2),
      '',
      '## Final worker output',
      '',
      workerOutput
    ].join('\n');

    await fs.writeFile(reportPath, report, 'utf8');

    if (status === 'ok') {
      process.exit(0);
    }

    if (attempt < maxAttempts) {
      console.log('\nVerifier reported remaining tasks; continuing to next attempt.\n');
    }
  }

  console.error(`Exhausted ${maxAttempts} attempts without STATUS: ok.`);
  process.exit(1);
}

function extractTitle(markdown) {
  const lines = markdown.split(/\r?\n/);
  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed.startsWith('# ')) {
      return trimmed.replace(/^#\s+/, '');
    }
  }
  return null;
}

async function runCodex(prompt, readOnly, model) {
  return new Promise((resolve, reject) => {
    const args = ['exec'];
    if (readOnly) {
      args.push('--sandbox', 'read-only');
    } else {
      args.push('--dangerously-bypass-approvals-and-sandbox');
    }
    args.push('--model', model);

    const child = spawn('codex', args, { stdio: ['pipe', 'pipe', 'pipe'] });

    let output = '';
    child.stdout.on('data', chunk => {
      const text = chunk.toString();
      process.stdout.write(text);
      output += text;
    });
    child.stderr.on('data', chunk => {
      const text = chunk.toString();
      process.stderr.write(text);
    });
    child.on('error', err => reject(err));
    child.on('close', code => {
      if (code !== 0) {
        reject(new Error(`codex exited with code ${code}`));
      } else {
        resolve(output);
      }
    });

    child.stdin.write(prompt);
    child.stdin.end();
  });
}

main().catch(err => {
  console.error('Unexpected error:', err);
  process.exit(1);
});
