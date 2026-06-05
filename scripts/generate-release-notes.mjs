#!/usr/bin/env node

import { execFileSync } from 'node:child_process'
import { existsSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const scriptDir = dirname(fileURLToPath(import.meta.url))
const repoRoot = resolve(scriptDir, '..')
const defaultTemplatePath = resolve(
  repoRoot,
  '.github/release-notes-template.md'
)

const args = parseArgs(process.argv.slice(2))
const tag = args.tag ?? env('RELEASE_TAG') ?? git(['describe', '--tags'])
const repo = args.repo ?? env('GITHUB_REPOSITORY') ?? env('GH_REPO')
const outputPath = args.output
const templatePath = args.template ?? defaultTemplatePath
const previousTag = args.from ?? findPreviousTag(tag)
const revisionRange = previousTag ? `${previousTag}..${tag}` : tag

const commits = gitLog(revisionRange)
const sections = groupCommits(commits, repo)
const fullChangelog = formatFullChangelog(repo, previousTag, tag)
const template = readTemplate(templatePath)
const releaseNotes = renderTemplate(template, {
  WHAT_CHANGED: formatSection(sections.whatChanged),
  BUG_FIX: formatSection(sections.bugFix),
  MAINTENANCE: formatSection(sections.maintenance),
  OTHER: formatSection(sections.other),
  FULL_CHANGELOG: fullChangelog,
})

if (outputPath) {
  writeFileSync(outputPath, releaseNotes)
} else {
  process.stdout.write(releaseNotes)
}

function parseArgs(values) {
  const parsed = {}

  for (let index = 0; index < values.length; index += 1) {
    const value = values[index]
    if (!value.startsWith('--')) {
      continue
    }

    const [rawKey, inlineValue] = value.slice(2).split('=', 2)
    const key = rawKey.replaceAll('-', '_')
    parsed[key] = inlineValue ?? values[index + 1]

    if (inlineValue === undefined) {
      index += 1
    }
  }

  return {
    from: parsed.from,
    output: parsed.output,
    repo: parsed.repo,
    tag: parsed.tag,
    template: parsed.template,
  }
}

function env(name) {
  return process.env[name] || undefined
}

function git(args) {
  return execFileSync('git', args, {
    cwd: repoRoot,
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'pipe'],
  }).trim()
}

function tryGit(args) {
  try {
    return git(args)
  } catch {
    return undefined
  }
}

function findPreviousTag(currentTag) {
  return tryGit([
    'describe',
    '--tags',
    '--abbrev=0',
    '--match',
    'v*',
    `${currentTag}^`,
  ])
}

function gitLog(range) {
  const output = tryGit([
    'log',
    '--reverse',
    '--format=%H%x1f%h%x1f%s%x1f%an%x1f%ae%x1e',
    range,
  ])

  if (!output) {
    return []
  }

  return output
    .split('\x1e')
    .map((entry) => entry.trim())
    .filter(Boolean)
    .map((entry) => {
      const [hash, shortHash, subject, authorName, authorEmail] =
        entry.split('\x1f')
      return {
        authorEmail,
        authorName,
        hash,
        shortHash,
        subject,
      }
    })
}

function groupCommits(commits, repo) {
  const sections = {
    whatChanged: [],
    bugFix: [],
    maintenance: [],
    other: [],
  }

  for (const commit of commits) {
    const item = formatCommit(commit, repo)
    const type = commitType(commit.subject)

    if (type === 'feat') {
      sections.whatChanged.push(item)
    } else if (type === 'fix') {
      sections.bugFix.push(item)
    } else if (isMaintenanceType(type)) {
      sections.maintenance.push(item)
    } else {
      sections.other.push(item)
    }
  }

  return sections
}

function commitType(subject) {
  const match = subject.match(/^([a-z]+)(?:\([^)]+\))?!?:\s/i)
  return match?.[1]?.toLowerCase()
}

function isMaintenanceType(type) {
  return [
    'build',
    'chore',
    'ci',
    'docs',
    'perf',
    'refactor',
    'style',
    'test',
  ].includes(type)
}

function formatCommit(commit, repo) {
  const hashText = repo
    ? `[\`${commit.shortHash}\`](https://github.com/${repo}/commit/${commit.hash})`
    : `\`${commit.shortHash}\``
  const author = formatAuthor(commit.authorName, commit.authorEmail)

  return `- ${hashText} ${commit.subject} by ${author}`
}

function formatAuthor(authorName, authorEmail) {
  const userName = githubUserFromEmail(authorEmail)

  if (userName) {
    return `[@${userName}](https://github.com/${userName})`
  }

  return escapeMarkdown(authorName || 'unknown')
}

function githubUserFromEmail(email) {
  const match = email?.match(/^(?:\d+\+)?([^@]+)@users\.noreply\.github\.com$/)
  return match?.[1]
}

function escapeMarkdown(value) {
  return value.replaceAll('[', '\\[').replaceAll(']', '\\]')
}

function formatSection(items) {
  return items.length > 0 ? items.join('\n') : '- No changes.'
}

function formatFullChangelog(repo, fromTag, toTag) {
  if (!repo || !fromTag) {
    return toTag
  }

  const range = `${fromTag}...${toTag}`
  return `[\`${range}\`](https://github.com/${repo}/compare/${fromTag}...${toTag})`
}

function readTemplate(path) {
  if (!existsSync(path)) {
    throw new Error(`Release notes template not found: ${path}`)
  }

  return readFileSync(path, 'utf8')
}

function renderTemplate(template, values) {
  return Object.entries(values).reduce(
    (content, [key, value]) => content.replaceAll(`{{${key}}}`, value),
    template
  )
}
