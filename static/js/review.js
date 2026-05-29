const reviewPageSize = 20;

async function showExamWrong() {
  if (!state.userId) return alert("请先选择账号。");
  const examId = els.examSelect.value || state.exam?.id;
  if (!examId) return alert("请先选择试卷。");
  await openPagedReview({
    title: "本套错题",
    emptyMessage: "本套试卷还没有错题。",
    returnMode: "setup",
    meta: (page) => `共 ${page.total || 0} 道，已加载 ${Math.min(page.offset + page.items.length, page.total || 0)} 道`,
    url: (offset) =>
      `/api/exams/wrong?userId=${encodeURIComponent(state.userId)}&examId=${encodeURIComponent(examId)}&offset=${offset}&limit=${reviewPageSize}`,
  });
}

async function showWrongBook() {
  if (!state.userId) return alert("请先选择账号。");
  await openPagedReview({
    title: "错题库",
    emptyMessage: "当前账号还没有错题。",
    returnMode: "setup",
    meta: (page) => `按错误次数优先排序，共 ${page.total || 0} 道，已加载 ${Math.min(page.offset + page.items.length, page.total || 0)} 道`,
    url: (offset) => `/api/wrong-book?userId=${encodeURIComponent(state.userId)}&offset=${offset}&limit=${reviewPageSize}`,
  });
}

async function showFeaturedWrongBook() {
  if (!state.userId) return alert("请先选择账号。");
  await openPagedReview({
    title: "精选错题集",
    emptyMessage: "当前账号还没有精选错题。",
    returnMode: "setup",
    meta: (page) => `错题次数 - 答对次数 >= -3，共 ${page.total || 0} 道，已加载 ${Math.min(page.offset + page.items.length, page.total || 0)} 道`,
    url: (offset) => `/api/featured-wrong-book?userId=${encodeURIComponent(state.userId)}&offset=${offset}&limit=${reviewPageSize}`,
  });
}

async function showDoubtBook() {
  if (!state.userId) return alert("请先选择账号。");
  const payload = await api(`/api/doubts?userId=${encodeURIComponent(state.userId)}`);
  const items = payload.items || [];
  if (!items.length) return alert("当前账号还没有疑问题。");
  renderReview("疑问库", "按标记时间排序", items, "setup");
}

async function openPagedReview(config) {
  const firstPage = await api(config.url(0));
  const items = firstPage.items || [];
  if (!items.length) return alert(config.emptyMessage);
  state.reviewPager = {
    config,
    items: [],
    nextOffset: 0,
    total: firstPage.total || 0,
    hasMore: false,
  };
  renderReviewFrame(config.title, config.meta(firstPage), config.returnMode);
  appendReviewPage(firstPage);
}

function renderReviewFrame(title, meta, returnMode = "setup") {
  state.reviewReturnMode = returnMode;
  els.setupStage.classList.add("hidden");
  els.questionStage.classList.add("hidden");
  els.reviewStage.classList.remove("hidden");
  els.reviewTitle.textContent = title;
  els.reviewMeta.textContent = meta;
  els.reviewList.innerHTML = "";
}

function appendReviewPage(page) {
  const pager = state.reviewPager;
  const items = page.items || [];
  removeReviewPagerButton();
  const startIndex = pager.items.length;
  items.forEach((item, index) => {
    els.reviewList.append(createWrongCard(item, startIndex + index));
  });
  pager.items.push(...items);
  pager.nextOffset = page.offset + items.length;
  pager.total = page.total || pager.total || pager.items.length;
  pager.hasMore = Boolean(page.hasMore);
  pager.config && (els.reviewMeta.textContent = pager.config.meta({
    ...page,
    offset: 0,
    items: pager.items,
    total: pager.total,
  }));
  if (pager.hasMore) appendReviewPagerButton();
}

function appendReviewPagerButton() {
  const footer = document.createElement("div");
  footer.className = "review-pager";
  footer.dataset.reviewPager = "true";
  const button = document.createElement("button");
  button.type = "button";
  button.className = "review-load-button";
  button.textContent = "加载更多";
  button.addEventListener("click", () => loadMoreReviewItems(button).catch((error) => alert(error.message)));
  footer.append(button);
  els.reviewList.append(footer);
}

function removeReviewPagerButton() {
  els.reviewList.querySelector('[data-review-pager="true"]')?.remove();
}

async function loadMoreReviewItems(button) {
  const pager = state.reviewPager;
  if (!pager?.hasMore) return;
  button.disabled = true;
  button.textContent = "加载中...";
  const page = await api(pager.config.url(pager.nextOffset));
  appendReviewPage(page);
}

function renderReview(title, meta, items, returnMode = "setup") {
  items = Array.isArray(items) ? items : [];
  renderReviewFrame(title, meta, returnMode);
  if (!items.length) {
    els.reviewList.innerHTML = `<div class="empty-copy compact-empty"><p>暂时没有错题。</p></div>`;
    return;
  }
  items.forEach((item, index) => {
    els.reviewList.append(createWrongCard(item, index));
  });
}

function createWrongCard(item, index) {
  const card = document.createElement("article");
  card.className = "wrong-card";
  const question = item.question || { options: [] };
  const wrong = item.wrong || {};
  const doubt = item.doubt || {};
  const answer = item.answer || {};
  const badge = wrong.wrongCount
    ? `错 ${wrong.wrongCount || 0} 次 / 对 ${wrong.correctCount || 0} 次`
    : "疑问标记";
  card.innerHTML = `
    <div class="wrong-meta">
      <span>#${index + 1}</span>
      <span>${escapeHtml(question.source || "")}</span>
      <strong>${escapeHtml(badge)}</strong>
      ${doubt.markedAt ? `<strong>疑问</strong>` : ""}
    </div>
    <h4>${escapeHtml(question.stem || "")}</h4>
    <div class="wrong-options">${(question.options || []).map((option) => `<p><b>${escapeHtml(option.label)}.</b> ${escapeHtml(option.text)}</p>`).join("")}</div>
    <p class="wrong-answer">正确答案：${escapeHtml(formatAnswer(question))}</p>
    ${answer.judged ? `<p class="wrong-answer">上次选择：${escapeHtml((answer.selected || []).join("、") || "未记录")}</p>` : ""}
    <p class="wrong-explain">${escapeHtml(question.explanation || "该题没有解析文本。")}</p>
  `;
  return card;
}
