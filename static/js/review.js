async function showExamWrong() {
  if (!state.userId) return alert("请先选择账号。");
  const examId = els.examSelect.value || state.exam?.id;
  if (!examId) return alert("请先选择试卷。");
  const payload = await api(`/api/exams/wrong?userId=${encodeURIComponent(state.userId)}&examId=${encodeURIComponent(examId)}`);
  const items = payload.items || [];
  if (!items.length) return alert("本套试卷还没有错题。");
  renderReview("本套错题", `${items.length} 道`, items, "setup");
}

async function showWrongBook() {
  if (!state.userId) return alert("请先选择账号。");
  const payload = await api(`/api/wrong-book?userId=${encodeURIComponent(state.userId)}`);
  const items = payload.items || [];
  if (!items.length) return alert("当前账号还没有错题。");
  renderReview("错题库", "按错误次数优先排序", items, "setup");
}

async function showFeaturedWrongBook() {
  if (!state.userId) return alert("请先选择账号。");
  const payload = await api(`/api/featured-wrong-book?userId=${encodeURIComponent(state.userId)}`);
  const items = payload.items || [];
  if (!items.length) return alert("当前账号还没有精选错题。");
  renderReview("精选错题集", "错题次数 - 答对次数 >= -3", items, "setup");
}

async function showDoubtBook() {
  if (!state.userId) return alert("请先选择账号。");
  const payload = await api(`/api/doubts?userId=${encodeURIComponent(state.userId)}`);
  const items = payload.items || [];
  if (!items.length) return alert("当前账号还没有疑问题。");
  renderReview("疑问库", "按标记时间排序", items, "setup");
}

function renderReview(title, meta, items, returnMode = "setup") {
  items = Array.isArray(items) ? items : [];
  state.reviewReturnMode = returnMode;
  els.setupStage.classList.add("hidden");
  els.questionStage.classList.add("hidden");
  els.reviewStage.classList.remove("hidden");
  els.reviewTitle.textContent = title;
  els.reviewMeta.textContent = meta;
  els.reviewList.innerHTML = "";
  if (!items.length) {
    els.reviewList.innerHTML = `<div class="empty-copy compact-empty"><p>暂时没有错题。</p></div>`;
    return;
  }
  items.forEach((item, index) => {
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
    els.reviewList.append(card);
  });
}
