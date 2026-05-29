async function init() {
  await Promise.all([loadUsers(), loadLibrary()]);
  await loadExamList();
  renderSetup();
}

els.userSelect.addEventListener("change", async () => {
  state.userId = els.userSelect.value;
  localStorage.setItem("examPaper.userId", state.userId);
  state.exam = null;
  state.questions = [];
  state.questionTotal = 0;
  state.loadedQuestionRanges = [];
  state.questionLoadPromises = new Map();
  state.answers = new Map();
  state.answeredIndexes = new Set();
  state.wrongIndexes = new Set();
  state.doubts = new Set();
  await loadExamList();
  if (state.incompleteExams.length) await continueExam(true);
  else showSetup();
});
els.userNameInput.addEventListener("input", updateRegisterButton);
els.registerButton.addEventListener("click", () => registerUser().catch((error) => alert(error.message)));
els.clearUserButton.addEventListener("click", () => clearUserData().catch((error) => alert(error.message)));
els.fileInput?.addEventListener("change", (event) => importFiles(event.target.files).catch((error) => alert(error.message)));
els.folderInput?.addEventListener("change", (event) => importFiles(event.target.files).catch((error) => alert(error.message)));
els.questionCountInput.addEventListener("input", () => {
  clampQuestionCount();
  updateSetupButtons();
});
els.maxQuestionButton.addEventListener("click", useMaxQuestionCount);
els.startExamButton.addEventListener("click", () => startExam().catch((error) => alert(error.message)));
els.continueExamButton.addEventListener("click", () => continueExam(false).catch((error) => alert(error.message)));
els.examSelect.addEventListener("change", updateSetupButtons);
els.deleteExamButton.addEventListener("click", () => deleteSelectedExam().catch((error) => alert(error.message)));
els.wrongExamButton.addEventListener("click", () => startWrongExam().catch((error) => alert(error.message)));
els.featuredWrongButton.addEventListener("click", () => startFeaturedWrongExam().catch((error) => alert(error.message)));
els.doubtExamButton.addEventListener("click", () => startDoubtExam().catch((error) => alert(error.message)));
els.examWrongButton.addEventListener("click", () => showExamWrong().catch((error) => alert(error.message)));
els.wrongBookButton.addEventListener("click", () => showWrongBook().catch((error) => alert(error.message)));
els.doubtBookButton.addEventListener("click", () => showDoubtBook().catch((error) => alert(error.message)));
els.submitButton.addEventListener("click", () => submitCurrent().catch((error) => alert(error.message)));
els.doubtButton.addEventListener("click", () => toggleDoubt().catch((error) => alert(error.message)));
els.backSetupButton.addEventListener("click", showSetup);
els.prevUnansweredButton.addEventListener("click", () => goUnanswered(-1));
els.prevButton.addEventListener("click", () => {
  goQuestion(Math.max(0, state.current - 1)).catch((error) => alert(error.message));
});
els.nextButton.addEventListener("click", () => {
  goQuestion(Math.min(questionTotal() - 1, state.current + 1)).catch((error) => alert(error.message));
});
els.nextUnansweredButton.addEventListener("click", () => goUnanswered(1));
els.closeReviewButton.addEventListener("click", () => {
  state.reviewPager = null;
  if (state.reviewReturnMode === "question" && state.exam && questionTotal()) showQuestion();
  else showSetup();
});

init().catch((error) => {
  console.error(error);
  setBusy(error.message);
});
