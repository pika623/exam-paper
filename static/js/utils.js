function answeredCount() {
  return state.answeredIndexes?.size || 0;
}

function currentUser() {
  return state.users.find((user) => user.id === state.userId);
}

function isMultiple(question) {
  const type = question.type || "";
  return type.includes("多") || question.answer.length > 1;
}

function isCorrect(selected, answer) {
  return normalizeChoice(selected).join("|") === normalizeChoice(answer).join("|");
}

function normalizeChoice(value) {
  return [...value].map(String).map((item) => item.toUpperCase()).sort();
}

function formatAnswer(question) {
  question = question || {};
  question.answer = question.answer || [];
  question.options = question.options || [];
  const labels = question.answer.length ? question.answer.join("、") : question.answerText;
  const optionTexts = question.options
    .filter((option) => question.answer.includes(option.label))
    .map((option) => `${option.label}. ${option.text}`);
  return optionTexts.length ? `${labels}（${optionTexts.join("；")}）` : labels;
}

function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}


