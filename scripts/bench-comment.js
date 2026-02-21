// bench-comment.js — Parse benchstat output into a glanceable PR comment.
// Called by bench-compare.yml with: node scripts/bench-comment.js .perf/base.txt .perf/pr.txt

const { execSync } = require('child_process');
const fs = require('fs');

const baseFile = process.argv[2];
const prFile = process.argv[3];
const outFile = process.argv[4] || '.perf/comment.md';

const raw = execSync(`benchstat ${baseFile} ${prFile} 2>&1`, { encoding: 'utf8' });

const lines = raw.split('\n');
const results = [];

for (const line of lines) {
  // benchstat output lines look like:
  // BenchmarkName-N   100.0n ± 5%   102.0n ± 3%   +2.00% (p=0.041)
  // BenchmarkName-N   100.0n ± 5%   99.0n ± 3%    ~ (p=0.310)
  const match = line.match(
    /^(\S+?)(?:-\d+)?\s+[\d.]+[nµm]?s?\s*±\s*\d+%\s+[\d.]+[nµm]?s?\s*±\s*\d+%\s+([~+-][\d.]*%?)\s*\(p=([\d.]+)[^)]*\)/
  );
  if (!match) continue;

  const name = match[1].replace(/^Benchmark/, '');
  const change = match[2].trim();
  const pValue = parseFloat(match[3]);

  let icon, verdict;
  if (change === '~') {
    icon = '~';
    verdict = 'no change';
  } else {
    const pct = parseFloat(change);
    if (isNaN(pct)) {
      icon = '~';
      verdict = 'no change';
    } else if (pct > 5 && pValue < 0.05) {
      icon = ':warning:';
      verdict = `**${change} slower**`;
    } else if (pct < -5 && pValue < 0.05) {
      icon = ':rocket:';
      verdict = `**${change} faster**`;
    } else {
      icon = ':white_check_mark:';
      verdict = 'within noise';
    }
  }

  results.push({ icon, name, change, pValue, verdict });
}

let body;

if (results.length === 0) {
  body = [
    '## Benchmark Comparison',
    '',
    'No benchmark results could be parsed. Raw output:',
    '',
    '```',
    raw.trim(),
    '```',
  ].join('\n');
} else {
  const hasRegression = results.some(r => r.icon === ':warning:');
  const hasImprovement = results.some(r => r.icon === ':rocket:');

  let summary;
  if (hasRegression) {
    summary = ':warning: **Performance regression detected**';
  } else if (hasImprovement) {
    summary = ':rocket: **Performance improved**';
  } else {
    summary = ':white_check_mark: **No significant performance change**';
  }

  const table = [
    '| | Benchmark | Delta | Verdict |',
    '|---|---|---|---|',
    ...results.map(r => `| ${r.icon} | \`${r.name}\` | ${r.change} (p=${r.pValue.toFixed(3)}) | ${r.verdict} |`),
  ].join('\n');

  body = [
    `## Benchmark Comparison`,
    '',
    summary,
    '',
    table,
    '',
    '<details>',
    '<summary>Raw benchstat output</summary>',
    '',
    '```',
    raw.trim(),
    '```',
    '',
    '</details>',
  ].join('\n');
}

fs.writeFileSync(outFile, body);
console.log(`Wrote comment to ${outFile}`);
