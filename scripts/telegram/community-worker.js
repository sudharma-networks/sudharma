'use strict';

const {
  parseCommunityUpdate,
  validateReportText,
  staticReply,
  hasReportMarker,
  buildReportIssue,
  canCreateReport,
} = require('./community-core.js');

const MAX_REPLIES_PER_RUN = 20;
const REPORT_WINDOW_MS = 60 * 60 * 1000;
const COMMUNITY_REPORT_MARKER_RE = /<!-- sudharma-telegram-community-report:v1 update_id=\d+ -->/;

function createTelegramAdapter({ token, fetchImpl = globalThis.fetch }) {
  if (typeof token !== 'string' || token.length === 0) {
    throw new Error('Telegram bot token is required');
  }
  if (typeof fetchImpl !== 'function') {
    throw new Error('A fetch implementation is required');
  }

  async function call(method, payload = {}) {
    let response;
    try {
      response = await fetchImpl(`https://api.telegram.org/bot${token}/${method}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
    } catch {
      throw new Error(`Telegram API ${method} failed`);
    }

    let data;
    try {
      data = await response.json();
    } catch {
      throw new Error(`Telegram API ${method} failed`);
    }

    if (!response.ok || !data || data.ok !== true) {
      throw new Error(`Telegram API ${method} failed`);
    }
    return data.result;
  }

  return {
    getMe() {
      return call('getMe');
    },
    getWebhookInfo() {
      return call('getWebhookInfo');
    },
    deleteWebhook({ dropPendingUpdates }) {
      return call('deleteWebhook', { drop_pending_updates: Boolean(dropPendingUpdates) });
    },
    getUpdates({ offset, limit, timeout } = {}) {
      const payload = {};
      if (offset !== undefined) payload.offset = offset;
      if (limit !== undefined) payload.limit = limit;
      if (timeout !== undefined) payload.timeout = timeout;
      return call('getUpdates', payload);
    },
    sendMessage(payload) {
      return call('sendMessage', payload);
    },
  };
}

function parseRepository(repository) {
  if (typeof repository !== 'string') throw new Error('GitHub repository is required');
  const parts = repository.split('/');
  if (parts.length !== 2 || parts.some((part) => !part)) {
    throw new Error('GitHub repository must be owner/name');
  }
  return parts.map(encodeURIComponent).join('/');
}

function createGithubAdapter({ token, repository, fetchImpl = globalThis.fetch }) {
  if (typeof token !== 'string' || token.length === 0) {
    throw new Error('GitHub token is required');
  }
  if (typeof fetchImpl !== 'function') {
    throw new Error('A fetch implementation is required');
  }
  const repositoryPath = parseRepository(repository);
  const baseUrl = `https://api.github.com/repos/${repositoryPath}`;

  async function requestJson(operation, url, options = {}) {
    let response;
    try {
      response = await fetchImpl(url, {
        ...options,
        headers: {
          Accept: 'application/vnd.github+json',
          Authorization: `Bearer ${token}`,
          'X-GitHub-Api-Version': '2022-11-28',
          ...(options.headers || {}),
        },
      });
    } catch {
      throw new Error(`GitHub API ${operation} failed`);
    }
    if (!response || !response.ok) {
      throw new Error(`GitHub API ${operation} failed`);
    }
    try {
      return await response.json();
    } catch {
      throw new Error(`GitHub API ${operation} failed`);
    }
  }

  return {
    async createIssue({ title, body }) {
      const data = await requestJson('create issue', `${baseUrl}/issues`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ title, body }),
      });
      if (!data || typeof data.html_url !== 'string' || data.html_url.length === 0) {
        throw new Error('GitHub API create issue failed');
      }
      return data;
    },

    async listRecentIssues({ since }) {
      const sinceDate = new Date(since);
      if (Number.isNaN(sinceDate.getTime())) {
        throw new Error('GitHub recent issue window is invalid');
      }

      const issues = [];
      for (let page = 1; page <= 100; page += 1) {
        const url = new URL(`${baseUrl}/issues`);
        url.searchParams.set('state', 'all');
        url.searchParams.set('since', sinceDate.toISOString());
        url.searchParams.set('per_page', '100');
        url.searchParams.set('sort', 'created');
        url.searchParams.set('direction', 'desc');
        url.searchParams.set('page', String(page));

        const data = await requestJson('list recent issues', url.toString(), { method: 'GET' });
        if (!Array.isArray(data)) {
          throw new Error('GitHub API list recent issues failed');
        }
        if (data.length === 0) return issues;

        for (const item of data) {
          if (!item || item.pull_request) continue;
          issues.push({
            html_url: item.html_url,
            body: typeof item.body === 'string' ? item.body : '',
            created_at: item.created_at,
          });
        }

        const oldestTime = Math.min(...data.map((item) => Date.parse(item && item.created_at)).filter(Number.isFinite));
        if (data.length < 100 || (Number.isFinite(oldestTime) && oldestTime < sinceDate.getTime())) {
          return issues;
        }
      }
      throw new Error('GitHub API list recent issues failed');
    },
  };
}

function replyPayload(message, text) {
  const payload = {
    chat_id: message.chat.id,
    text,
    reply_parameters: { message_id: message.message_id },
    link_preview_options: { is_disabled: true },
  };
  if (Number.isInteger(message.message_thread_id)) {
    payload.message_thread_id = message.message_thread_id;
  }
  return payload;
}

function isRecentCommunityReport(issue, sinceMs) {
  if (!issue || typeof issue.body !== 'string' || !COMMUNITY_REPORT_MARKER_RE.test(issue.body)) {
    return false;
  }
  const createdMs = Date.parse(issue.created_at);
  return Number.isFinite(createdMs) && createdMs >= sinceMs;
}

function createWorker({ telegram, github, now = () => new Date(), logger = console }) {
  if (!telegram || typeof telegram.getMe !== 'function' || typeof telegram.getUpdates !== 'function' || typeof telegram.sendMessage !== 'function') {
    throw new Error('Telegram dependency is invalid');
  }
  if (!github || typeof github !== 'object') {
    throw new Error('GitHub dependency is invalid');
  }
  if (typeof now !== 'function') {
    throw new Error('now must be a function');
  }

  async function poll() {
    const bot = await telegram.getMe();
    if (!bot || typeof bot.username !== 'string' || bot.username.length === 0) {
      throw new Error('Telegram bot username is unavailable');
    }

    const fetched = await telegram.getUpdates({ limit: 100, timeout: 0 });
    if (!Array.isArray(fetched)) {
      throw new Error('Telegram getUpdates result must be an array');
    }

    const updates = [...fetched].sort((left, right) => left.update_id - right.update_id);
    let replies = 0;
    let createdReports = 0;
    let recentIssues = null;
    let reportWindowStart = null;
    let lastConfirmed = null;

    function ensureReplyBudget() {
      if (replies >= MAX_REPLIES_PER_RUN) {
        throw new Error('Telegram community reply limit reached');
      }
    }

    async function sendReply(message, text) {
      ensureReplyBudget();
      await telegram.sendMessage(replyPayload(message, text));
      replies += 1;
    }

    async function loadRecentIssues() {
      if (recentIssues !== null) return recentIssues;
      const current = now();
      const currentDate = current instanceof Date ? current : new Date(current);
      if (Number.isNaN(currentDate.getTime())) {
        throw new Error('Current time is invalid');
      }
      reportWindowStart = currentDate.getTime() - REPORT_WINDOW_MS;
      if (typeof github.listRecentIssues !== 'function') {
        throw new Error('GitHub recent issue listing is unavailable');
      }
      recentIssues = await github.listRecentIssues({ since: new Date(reportWindowStart).toISOString() });
      if (!Array.isArray(recentIssues)) {
        throw new Error('GitHub recent issue listing is invalid');
      }
      return recentIssues;
    }

    async function handleReport(update, action) {
      let reportText;
      try {
        reportText = validateReportText(action.args);
      } catch {
        await sendReply(action.message, staticReply('report-usage'));
        return;
      }

      // A report must not create a public GitHub issue if this run cannot
      // also return its URL to the Telegram user.
      ensureReplyBudget();

      const issues = await loadRecentIssues();
      const existing = issues.find((issue) => hasReportMarker(issue.body, update.update_id));
      if (existing && typeof existing.html_url === 'string' && existing.html_url.length > 0) {
        await sendReply(action.message, `Report already tracked: ${existing.html_url}`);
        return;
      }

      const createdLastHour = issues.filter((issue) => isRecentCommunityReport(issue, reportWindowStart)).length;
      const decision = canCreateReport({ createdThisRun: createdReports, createdLastHour });
      if (!decision.allowed) {
        await sendReply(action.message, staticReply('throttle'));
        return;
      }

      if (typeof github.createIssue !== 'function') {
        throw new Error('GitHub issue creation is unavailable');
      }
      const issue = buildReportIssue({
        updateId: update.update_id,
        reportText,
        message: action.message,
        now: now(),
      });
      const created = await github.createIssue(issue);
      if (!created || typeof created.html_url !== 'string' || created.html_url.length === 0) {
        throw new Error('GitHub issue creation returned no URL');
      }

      createdReports += 1;
      recentIssues.push({
        html_url: created.html_url,
        body: typeof created.body === 'string' ? created.body : issue.body,
        created_at: typeof created.created_at === 'string' ? created.created_at : new Date().toISOString(),
      });

      await sendReply(
        action.message,
        `Report created: ${created.html_url}\nThe submitted report text is public on GitHub. Attachments remain in Telegram.`,
      );
    }

    async function handleUpdate(update) {
      const action = parseCommunityUpdate(update, bot.username);
      if (action.kind === 'ignore') return;

      if (action.kind === 'welcome') {
        await sendReply(action.message, staticReply('welcome'));
        return;
      }

      if (action.kind !== 'command') return;

      if (action.command === 'report') {
        await handleReport(update, action);
        return;
      }

      await sendReply(action.message, staticReply(action.command));
    }

    for (const update of updates) {
      if (!update || !Number.isInteger(update.update_id) || update.update_id < 0) {
        if (logger && typeof logger.error === 'function') {
          logger.error('Telegram community update has an invalid update_id');
        }
        break;
      }
      try {
        await handleUpdate(update);
        lastConfirmed = update.update_id;
      } catch {
        if (logger && typeof logger.error === 'function') {
          logger.error('Telegram community update processing failed', { updateId: update.update_id });
        }
        break;
      }
    }

    if (lastConfirmed !== null) {
      await telegram.getUpdates({ offset: lastConfirmed + 1, limit: 1, timeout: 0 });
    }

    return { lastConfirmed, replies, createdReports };
  }

  return { poll };
}

module.exports = {
  MAX_REPLIES_PER_RUN,
  createTelegramAdapter,
  createGithubAdapter,
  createWorker,
};
