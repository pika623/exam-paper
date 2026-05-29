async function loadExamList() {
  if (!state.userId) {
    state.incompleteExams = [];
    renderExamSelect();
    return;
  }
  const payload = await api(`/api/exams/list?userId=${encodeURIComponent(state.userId)}`);
  state.incompleteExams = payload.exams || [];
  renderExamSelect();
}

function renderExamSelect() {
  els.examSelect.innerHTML = "";
  if (!state.incompleteExams.length) {
    els.examSelect.innerHTML = `<option value="">没有未完成试卷</option>`;
    updateSetupButtons();
    return;
  }
  state.incompleteExams.forEach((exam) => {
    const option = document.createElement("option");
    option.value = exam.id;
    option.textContent = `${exam.title}（${exam.answered}/${exam.total}，错 ${exam.wrong || 0}）`;
    els.examSelect.append(option);
  });
  updateSetupButtons();
}

function updateSetupButtons() {
  const hasIncomplete = state.incompleteExams.length > 0;
  const current = currentUser();
  const selectedExam = selectedExamSummary();
  const hasSource = state.selectedSources.size > 0;
  const hasAvailableQuestions = state.selectedQuestionMax > 0;
  const requestedCount = Number(els.questionCountInput.value || 0);
  els.startExamButton.disabled = !hasSource || !hasAvailableQuestions || requestedCount <= 0;
  els.continueExamButton.disabled = !hasIncomplete;
  els.examWrongButton.disabled = !hasIncomplete || !selectedExamWrongCount();
  els.deleteExamButton.disabled = !hasIncomplete;
  els.maxQuestionButton.disabled = !hasSource || !hasAvailableQuestions;
  els.wrongExamButton.disabled = !(current?.wrongBookCount);
  els.featuredWrongButton.disabled = !(current?.featuredWrongCount);
  els.doubtExamButton.disabled = !(current?.doubtCount);
  els.wrongBookButton.disabled = !(current?.wrongBookCount);
  els.doubtBookButton.disabled = !(current?.doubtCount);
  els.wrongExamButton.textContent = `刷错题（${current?.wrongBookCount || 0}）`;
  els.featuredWrongButton.textContent = `刷精选错题（${current?.featuredWrongCount || 0}）`;
  els.doubtExamButton.textContent = `刷疑问题（${current?.doubtCount || 0}）`;
  els.examWrongButton.textContent = `本套错题（${selectedExamWrongCount()}）`;
  els.wrongBookButton.textContent = `错题库（${current?.wrongBookCount || 0}）`;
  els.doubtBookButton.textContent = `疑问库（${current?.doubtCount || 0}）`;
}

function selectedExamSummary() {
  const selected = els.examSelect.value;
  return state.incompleteExams.find((exam) => exam.id === selected);
}

function selectedExamWrongCount() {
  if (state.exam && state.exam.id === els.examSelect.value) return currentExamWrongCount();
  return selectedExamSummary()?.wrong || 0;
}

function currentExamWrongCount() {
  return state.wrongIndexes?.size || 0;
}

function questionTotal() {
  return state.questionTotal || state.questions.length || 0;
}

function resetQuestionCache(total = 0) {
  state.questionTotal = total;
  state.questions = Array.from({ length: total });
  state.loadedQuestionRanges = [];
  state.questionLoadPromises = new Map();
}

function mergeQuestionWindow(questions, offset = 0) {
  questions = Array.isArray(questions) ? questions : [];
  questions.forEach((question, index) => {
    state.questions[offset + index] = question;
  });
  if (questions.length) state.loadedQuestionRanges.push([offset, offset + questions.length - 1]);
}

function mergeAnswers(answers) {
  Object.entries(answers || {}).forEach(([questionId, answer]) => {
    state.answers.set(questionId, answer);
  });
}

async function ensureQuestionLoaded(index) {
  if (!state.userId || !state.exam || index < 0 || index >= questionTotal()) return;
  if (state.questions[index]) return;
  const key = String(Math.floor(index / 25));
  if (!state.questionLoadPromises.has(key)) {
    const url = `/api/exams/questions?userId=${encodeURIComponent(state.userId)}&examId=${encodeURIComponent(state.exam.id)}&center=${index}&limit=25`;
    state.questionLoadPromises.set(
      key,
      api(url)
        .then((payload) => {
          mergeQuestionWindow(payload.questions || [], Number(payload.offset || 0));
          mergeAnswers(payload.answers || {});
          state.questionTotal = Number(payload.total || state.questionTotal || 0);
        })
        .finally(() => state.questionLoadPromises.delete(key))
    );
  }
  await state.questionLoadPromises.get(key);
}

async function updateSelectedQuestionMax() {
  const requestId = state.questionMaxRequest + 1;
  state.questionMaxRequest = requestId;
  const sources = [...state.selectedSources];
  state.selectedQuestionMax = 0;
  els.questionCountInput.removeAttribute("max");
  els.questionMaxHint.textContent = sources.length ? "正在统计所选题库题量..." : "选择题库来源后显示最大题量";
  updateSetupButtons();

  if (!sources.length) return;
  const payload = await api("/api/questions/count", {
    method: "POST",
    body: JSON.stringify({ sources }),
  });
  if (requestId !== state.questionMaxRequest) return;
  state.selectedQuestionMax = Number(payload.count || 0);
  els.questionCountInput.max = String(state.selectedQuestionMax);
  if (state.selectedQuestionMax > 0) {
    els.questionCountInput.value = String(state.selectedQuestionMax);
  } else {
    clampQuestionCount();
  }
  els.questionMaxHint.textContent = state.selectedQuestionMax
    ? `当前选择最多 ${state.selectedQuestionMax} 题`
    : "所选题库没有可用题目";
  updateSetupButtons();
}

function clampQuestionCount() {
  const max = state.selectedQuestionMax;
  let count = Number(els.questionCountInput.value || 0);
  if (!Number.isFinite(count) || count < 1) count = 1;
  if (max > 0 && count > max) count = max;
  els.questionCountInput.value = String(count);
  return count;
}

function useMaxQuestionCount() {
  if (state.selectedQuestionMax <= 0) return;
  els.questionCountInput.value = String(state.selectedQuestionMax);
  updateSetupButtons();
}

async function startExam() {
  if (!state.userId) return alert("请先注册或选择账号。");
  const sources = [...state.selectedSources];
  if (!sources.length) return alert("请选择至少一个题库来源。");
  const count = clampQuestionCount();
  const payload = await api("/api/exams", {
    method: "POST",
    body: JSON.stringify({ userId: state.userId, sources, count }),
  });
  loadExamPayload(payload);
  await loadUsers();
  await loadExamList();
  showQuestion();
}

async function startWrongExam() {
  if (!state.userId) return alert("请先注册或选择账号。");
  const current = currentUser();
  if (!current || !current.wrongBookCount) return alert("当前账号还没有错题。");
  const payload = await api("/api/exams/wrong-practice", {
    method: "POST",
    body: JSON.stringify({ userId: state.userId, count: 0 }),
  });
  loadExamPayload(payload);
  await loadUsers();
  await loadExamList();
  showQuestion();
}

async function startFeaturedWrongExam() {
  if (!state.userId) return alert("请先注册或选择账号。");
  const current = currentUser();
  if (!current || !current.featuredWrongCount) return alert("当前账号还没有精选错题。");
  const payload = await api("/api/exams/featured-wrong-practice", {
    method: "POST",
    body: JSON.stringify({ userId: state.userId, count: 0 }),
  });
  loadExamPayload(payload);
  await loadUsers();
  await loadExamList();
  showQuestion();
}

async function startDoubtExam() {
  if (!state.userId) return alert("请先注册或选择账号。");
  const current = currentUser();
  if (!current || !current.doubtCount) return alert("当前账号还没有疑问题。");
  const payload = await api("/api/exams/doubt-practice", {
    method: "POST",
    body: JSON.stringify({ userId: state.userId, count: 0 }),
  });
  loadExamPayload(payload);
  await loadUsers();
  await loadExamList();
  showQuestion();
}

async function continueExam(silent = false) {
  if (!state.userId) {
    if (!silent) alert("请先选择账号。");
    return;
  }
  const selectedExam = els.examSelect.value;
  if (!state.incompleteExams.length || !selectedExam) {
    if (!silent) alert("当前账号没有未完成试卷。");
    return;
  }
  const query = selectedExam
    ? `userId=${encodeURIComponent(state.userId)}&examId=${encodeURIComponent(selectedExam)}`
    : `userId=${encodeURIComponent(state.userId)}`;
  const payload = await api(`/api/exams/current?${query}`);
  if (!payload.exam) {
    if (!silent) alert("当前账号没有未完成试卷。");
    return;
  }
  loadExamPayload(payload);
  await loadExamList();
  showQuestion();
}

async function deleteSelectedExam() {
  if (!state.userId) return alert("请先选择账号。");
  const examId = els.examSelect.value;
  if (!state.incompleteExams.length || !examId) return alert("当前账号没有可删除的未完成试卷。");
  const selected = state.incompleteExams.find((exam) => exam.id === examId);
  if (!confirm(`确认删除试卷“${selected?.title || examId}”？`)) return;
  await api(`/api/exams?userId=${encodeURIComponent(state.userId)}&examId=${encodeURIComponent(examId)}`, {
    method: "DELETE",
  });
  if (state.exam?.id === examId) {
    state.exam = null;
    state.questions = [];
    state.questionTotal = 0;
    state.loadedQuestionRanges = [];
    state.questionLoadPromises = new Map();
    state.answers = new Map();
    state.answeredIndexes = new Set();
    state.wrongIndexes = new Set();
    state.doubts = new Set();
    state.current = 0;
  }
  await loadUsers();
  await loadExamList();
  showSetup();
}

function loadExamPayload(payload) {
  state.exam = payload.exam;
  const total = Number(payload.total || payload.questions?.length || 0);
  resetQuestionCache(total);
  mergeQuestionWindow(payload.questions || [], Number(payload.offset || 0));
  state.answers = new Map();
  mergeAnswers(payload.answers || {});
  state.answeredIndexes = new Set(payload.answeredIndexes || []);
  state.wrongIndexes = new Set(payload.wrongIndexes || []);
  state.doubts = new Set(payload.doubts || []);
  state.current = Math.min(state.exam.current || 0, Math.max(0, questionTotal() - 1));
  renderQuestion();
  renderStats();
}

function applyExamState(exam) {
  if (!exam || !state.exam || exam.id !== state.exam.id) return;
  if (Number.isInteger(exam.current)) state.exam.current = exam.current;
  if (typeof exam.completed === "boolean") state.exam.completed = exam.completed;
  if (exam.updatedAt) state.exam.updatedAt = exam.updatedAt;
}

function showSetup() {
  els.setupStage.classList.remove("hidden");
  els.questionStage.classList.add("hidden");
  els.reviewStage.classList.add("hidden");
  renderSetup();
}

function showQuestion() {
  els.setupStage.classList.add("hidden");
  els.questionStage.classList.remove("hidden");
  els.reviewStage.classList.add("hidden");
  renderQuestion();
}

function renderSetup() {
  const current = currentUser();
  const selectedCount = state.selectedSources.size;
  els.sourceLabel.textContent = current ? `当前账号：${current.name}` : "未选择账号";
  els.sessionTitle.textContent = selectedCount ? `已选择 ${selectedCount} 个题库来源` : "选择账号和题库后开始";
  if (!selectedCount && state.incompleteExams.length) {
    els.sessionTitle.textContent = `有 ${state.incompleteExams.length} 套未完成试卷`;
  }
  renderAccountStats();
  updateSetupButtons();
}

function renderQuestion() {
  const total = questionTotal();
  if (!total || !state.exam) return;
  const question = state.questions[state.current];
  if (!question) {
    els.sourceLabel.textContent = "正在加载";
    els.sessionTitle.textContent = `${state.exam.title}${state.exam.completed ? "（已完成）" : ""}`;
    els.questionIndex.textContent = `第 ${state.current + 1} / ${total} 题`;
    els.questionType.textContent = "加载中";
    els.questionStem.textContent = "正在加载题目...";
    els.optionList.innerHTML = "";
    els.multiActions.classList.add("hidden");
    els.submitButton.disabled = true;
    els.resultCard.classList.add("hidden");
    els.prevButton.disabled = state.current === 0;
    els.nextButton.disabled = state.current === total - 1;
    els.prevUnansweredButton.disabled = findUnanswered(-1) < 0;
    els.nextUnansweredButton.disabled = findUnanswered(1) < 0;
    renderStats();
    ensureQuestionLoaded(state.current).then(renderQuestion).catch((error) => alert(error.message));
    return;
  }
  const saved = state.answers.get(question.id);
  const selected = saved?.selected || [];
  const judged = Boolean(saved?.judged);
  const isMulti = isMultiple(question);

  els.sourceLabel.textContent = question.source || "当前试卷";
  els.sessionTitle.textContent = `${state.exam.title}${state.exam.completed ? "（已完成）" : ""}`;
  els.questionIndex.textContent = `第 ${state.current + 1} / ${total} 题`;
  els.questionType.textContent = question.type || (isMulti ? "多选题" : "单选题");
  els.questionStem.textContent = question.stem;
  els.doubtButton.textContent = state.doubts.has(question.id) ? "已标记疑问" : "标记疑问";
  els.doubtButton.classList.toggle("active", state.doubts.has(question.id));
  els.progressBar.style.width = `${((answeredCount() || state.current + 1) / total) * 100}%`;
  els.optionList.innerHTML = "";

  question.options.forEach((option) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "option-button";
    button.dataset.label = option.label;
    button.innerHTML = `
      <span class="option-label">${escapeHtml(option.label)}</span>
      <span class="option-text">${escapeHtml(option.text)}</span>
    `;
    if (selected.includes(option.label)) button.classList.add("selected");
    if (judged) {
      if (question.answer.includes(option.label)) button.classList.add("correct");
      if (selected.includes(option.label) && !question.answer.includes(option.label)) button.classList.add("wrong");
    }
    button.disabled = judged;
    button.addEventListener("click", () => chooseOption(option.label));
    els.optionList.append(button);
  });

  els.multiActions.classList.toggle("hidden", !isMulti || judged);
  els.submitButton.disabled = !selected.length;
  els.resultCard.classList.toggle("hidden", !judged);
  if (judged) renderResult(question, saved);

  els.prevButton.disabled = state.current === 0;
  els.nextButton.disabled = state.current === total - 1;
  els.prevUnansweredButton.disabled = findUnanswered(-1) < 0;
  els.nextUnansweredButton.disabled = findUnanswered(1) < 0;
  renderStats();
  updateSetupButtons();
}

function renderAccountStats() {
  const current = currentUser();
  els.totalCount.textContent = current?.answeredTotal || 0;
  els.correctCount.textContent = current?.correctTotal || 0;
  els.wrongCount.textContent = current?.wrongTotal || 0;
  els.wrongBookStat?.classList.add("hidden");
}

function chooseOption(label) {
  const question = state.questions[state.current];
  const saved = state.answers.get(question.id) || { selected: [], judged: false };
  if (saved.judged) return;
  if (isMultiple(question)) {
    saved.selected = saved.selected.includes(label)
      ? saved.selected.filter((item) => item !== label)
      : [...saved.selected, label].sort();
    state.answers.set(question.id, saved);
    renderQuestion();
    return;
  }
  saveCurrentAnswer([label]);
}

async function submitCurrent() {
  const question = state.questions[state.current];
  const saved = state.answers.get(question.id);
  if (!saved || !saved.selected.length) return;
  await saveCurrentAnswer(saved.selected);
}

async function saveCurrentAnswer(selected) {
  const question = state.questions[state.current];
  const correct = isCorrect(selected, question.answer);
  const localAnswer = { selected: normalizeChoice(selected), correct, judged: true };
  state.answers.set(question.id, localAnswer);
  state.answeredIndexes.add(state.current);
  if (correct) state.wrongIndexes.delete(state.current);
  else state.wrongIndexes.add(state.current);
  renderQuestion();
  const payload = await api("/api/exams/answer", {
    method: "POST",
    body: JSON.stringify({
      userId: state.userId,
      examId: state.exam.id,
      questionId: question.id,
      selected,
      current: state.current,
    }),
  });
  applyExamState(payload.exam);
  state.answers.set(question.id, payload.answer);
  if (payload.answer?.judged) {
    state.answeredIndexes.add(state.current);
    if (payload.answer.correct) state.wrongIndexes.delete(state.current);
    else state.wrongIndexes.add(state.current);
  }
  await loadUsers();
  await loadExamList();
  renderQuestion();
}

async function toggleDoubt() {
  if (!state.userId || !state.exam || !questionTotal()) return;
  const question = state.questions[state.current];
  const marked = !state.doubts.has(question.id);
  if (marked) state.doubts.add(question.id);
  else state.doubts.delete(question.id);
  renderQuestion();
  try {
    await api("/api/doubts/toggle", {
      method: "POST",
      body: JSON.stringify({ userId: state.userId, questionId: question.id, marked }),
    });
    await loadUsers();
    updateSetupButtons();
    renderStats();
  } catch (error) {
    if (marked) state.doubts.delete(question.id);
    else state.doubts.add(question.id);
    renderQuestion();
    throw error;
  }
}

function findUnanswered(direction) {
  for (let index = state.current + direction; index >= 0 && index < questionTotal(); index += direction) {
    if (!state.answeredIndexes.has(index)) return index;
  }
  return -1;
}

function goUnanswered(direction) {
  const index = findUnanswered(direction);
  if (index < 0) return alert(direction > 0 ? "后面没有未作题。" : "前面没有未作题。");
  goQuestion(index);
}

async function goQuestion(index) {
  const total = questionTotal();
  if (!total) return;
  index = Math.max(0, Math.min(total - 1, index));
  state.current = index;
  if (state.exam) state.exam.current = state.current;
  renderQuestion();
  await ensureQuestionLoaded(index);
  saveProgress();
  renderQuestion();
}

function renderResult(question, saved) {
  els.resultCard.classList.toggle("wrong", !saved.correct);
  els.resultTitle.textContent = saved.correct ? "回答正确" : "回答错误";
  els.answerBadge.textContent = `答案：${formatAnswer(question)}`;
  els.explanationText.textContent = question.explanation || "该题没有解析文本。";
}

function renderStats() {
  els.wrongBookStat?.classList.remove("hidden");
  els.totalCount.textContent = questionTotal();
  const wrong = state.wrongIndexes?.size || 0;
  const answered = answeredCount();
  els.correctCount.textContent = Math.max(0, answered - wrong);
  els.wrongCount.textContent = wrong;
  els.wrongBookCount.textContent = currentUser()?.wrongBookCount || 0;
}

function saveProgress() {
  if (!state.userId || !state.exam) return;
  api("/api/exams/progress", {
    method: "POST",
    body: JSON.stringify({ userId: state.userId, examId: state.exam.id, current: state.current }),
  })
    .then((payload) => {
      applyExamState(payload.exam);
    })
    .catch((error) => console.warn(error));
}

