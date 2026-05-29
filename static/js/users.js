async function loadUsers() {
  const payload = await api("/api/users");
  state.users = payload.users || [];
  if (!state.users.some((user) => user.id === state.userId)) {
    state.userId = state.users[0]?.id || "";
  }
  if (state.userId) localStorage.setItem("examPaper.userId", state.userId);
  renderUsers();
}

function renderUsers() {
  els.userSelect.innerHTML = "";
  if (!state.users.length) {
    els.userSelect.innerHTML = `<option value="">请注册账号</option>`;
  } else {
    state.users.forEach((user) => {
      const option = document.createElement("option");
      option.value = user.id;
      option.textContent = `${user.name}${user.currentExamId ? "（有未完成）" : ""}`;
      option.selected = user.id === state.userId;
      els.userSelect.append(option);
    });
  }
  const current = currentUser();
  els.wrongBookCount.textContent = current?.wrongBookCount || 0;
  updateRegisterButton();
}

function updateRegisterButton() {
  const name = els.userNameInput.value.trim();
  const duplicated = state.users.some((user) => user.name.toLowerCase() === name.toLowerCase());
  els.registerButton.disabled = !name || duplicated;
  els.registerButton.textContent = duplicated && name ? "账号已存在" : "注册账号";
}

async function registerUser() {
  const name = els.userNameInput.value.trim();
  if (!name) return alert("请输入账号名称。");
  if (state.users.some((user) => user.name.toLowerCase() === name.toLowerCase())) {
    updateRegisterButton();
    return alert("账号名称已存在。");
  }
  const payload = await api("/api/users", {
    method: "POST",
    body: JSON.stringify({ name }),
  });
  state.userId = payload.user.id;
  localStorage.setItem("examPaper.userId", state.userId);
  els.userNameInput.value = "";
  await loadUsers();
  await loadExamList();
  renderSetup();
}

async function clearUserData() {
  if (!state.userId) return alert("请先选择账号。");
  if (!confirm("确认清空当前账号的练习进度和错题库？")) return;
  await api("/api/users/clear", {
    method: "POST",
    body: JSON.stringify({ userId: state.userId }),
  });
  state.exam = null;
  state.questions = [];
  state.answers = new Map();
  state.current = 0;
  await loadUsers();
  await loadExamList();
  showSetup();
}

