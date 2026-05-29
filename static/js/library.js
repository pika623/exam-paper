async function loadLibrary() {
  const payload = await api("/api/library");
  state.library = payload.files || [];
  renderLibrary();
  setBusy(state.library.length ? "选择来源后组卷" : "等待导入 PDF");
}

function renderLibrary() {
  els.libraryCount.textContent = state.library.length;
  els.libraryList.innerHTML = "";
  if (!state.library.length) {
    els.libraryList.innerHTML = `<p class="library-path">当前目录还没有 PDF 或 DOCX。</p>`;
    return;
  }
  const { treeItems, looseItems } = splitLibraryItems();
  if (treeItems.length) {
    els.libraryList.append(renderLibraryNode(buildLibraryTree(treeItems)));
  }
  if (looseItems.length) {
    const loose = document.createElement("section");
    loose.className = "loose-files";
    loose.innerHTML = `
      <div class="loose-head">
        <span>导入文件</span>
        <small>${looseItems.length}</small>
      </div>
      <div class="loose-list"></div>
    `;
    const list = loose.querySelector(".loose-list");
    looseItems
      .sort((a, b) => a.name.localeCompare(b.name, "zh-Hans-CN"))
      .forEach((item) => list.append(createLibraryFileCheck(item)));
    els.libraryList.append(loose);
  }
}

function createLibraryFileCheck(item) {
  const label = document.createElement("label");
  label.className = "library-check tree-file";
  label.innerHTML = `
      <input type="checkbox" value="${escapeHtml(item.path)}" />
      <span>
        <span class="library-name">${escapeHtml(item.name)}</span>
        <span class="library-path">${escapeHtml(item.path)}</span>
      </span>
      <span class="file-pill">${escapeHtml(item.suffix.replace(".", ""))}</span>
    `;
  const input = label.querySelector("input");
  input.checked = state.selectedSources.has(item.path);
  input.addEventListener("change", () => {
    if (input.checked) state.selectedSources.add(item.path);
    else state.selectedSources.delete(item.path);
    renderLibrary();
    renderSetup();
    updateSelectedQuestionMax().catch((error) => {
      els.questionMaxHint.textContent = error.message;
      updateSetupButtons();
    });
  });
  return label;
}

function splitLibraryItems() {
  const looseItems = [];
  const treeItems = [];
  state.library.forEach((item) => {
    if (item.path.startsWith("data/uploads/")) looseItems.push(item);
    else treeItems.push(item);
  });
  return { treeItems, looseItems };
}

function buildLibraryTree(items) {
  const root = { name: "题库项目", path: "题库项目", children: new Map(), files: [] };
  items.forEach((item) => {
    const parts = displayTreePath(item).split("/");
    const fileName = parts.pop();
    let node = root;
    const pathParts = [];
    parts.forEach((part) => {
      pathParts.push(part);
      if (!node.children.has(part)) {
        node.children.set(part, { name: part, path: pathParts.join("/"), children: new Map(), files: [] });
      }
      node = node.children.get(part);
    });
    node.files.push({ ...item, name: fileName || item.name });
  });
  return root;
}

function displayTreePath(item) {
  if (item.path.startsWith("data/imports/")) return item.path.slice("data/imports/".length);
  return item.path;
}

function renderLibraryNode(node) {
  const wrapper = document.createElement("div");
  wrapper.className = "tree-node";
  const files = collectNodeFiles(node);
  const nodeId = `node-${Math.random().toString(36).slice(2)}`;
  const children = sortedChildren(node);
  const isRoot = node.name === "题库项目";
  wrapper.innerHTML = `
    <div class="tree-row ${isRoot ? "tree-root" : ""}">
      <input id="${nodeId}" type="checkbox" />
      <button class="tree-toggle" type="button" aria-expanded="true">
        <span class="tree-caret">⌄</span>
        <span class="tree-folder">${escapeHtml(node.name)}</span>
      </button>
      <span class="tree-count">${files.length}</span>
    </div>
    <div class="tree-children"></div>
  `;
  const input = wrapper.querySelector("input");
  const toggle = wrapper.querySelector(".tree-toggle");
  const childrenWrap = wrapper.querySelector(".tree-children");
  const collapsed = state.collapsedLibraryNodes.has(node.path);
  childrenWrap.classList.toggle("collapsed", collapsed);
  toggle.setAttribute("aria-expanded", String(!collapsed));
  toggle.querySelector(".tree-caret").textContent = collapsed ? "›" : "⌄";
  updateTreeInput(input, files);
  toggle.addEventListener("click", () => {
    childrenWrap.classList.toggle("collapsed");
    const isCollapsed = childrenWrap.classList.contains("collapsed");
    if (isCollapsed) state.collapsedLibraryNodes.add(node.path);
    else state.collapsedLibraryNodes.delete(node.path);
    toggle.setAttribute("aria-expanded", String(!isCollapsed));
    toggle.querySelector(".tree-caret").textContent = isCollapsed ? "›" : "⌄";
  });
  input.addEventListener("change", () => {
    files.forEach((item) => {
      if (input.checked) state.selectedSources.add(item.path);
      else state.selectedSources.delete(item.path);
    });
    renderLibrary();
    renderSetup();
    updateSelectedQuestionMax().catch((error) => {
      els.questionMaxHint.textContent = error.message;
      updateSetupButtons();
    });
  });
  children.forEach((child) => childrenWrap.append(renderLibraryNode(child)));
  node.files
    .sort((a, b) => a.name.localeCompare(b.name, "zh-Hans-CN"))
    .forEach((item) => childrenWrap.append(createLibraryFileCheck(item)));
  return wrapper;
}

function sortedChildren(node) {
  return [...node.children.values()].sort((a, b) => a.name.localeCompare(b.name, "zh-Hans-CN"));
}

function collectNodeFiles(node) {
  const files = [...node.files];
  node.children.forEach((child) => files.push(...collectNodeFiles(child)));
  return files;
}

function updateTreeInput(input, items) {
  const selectedCount = items.filter((item) => state.selectedSources.has(item.path)).length;
  input.checked = items.length > 0 && selectedCount === items.length;
  input.indeterminate = selectedCount > 0 && selectedCount < items.length;
}

async function importFiles(files) {
  if (!files.length) return;
  const form = new FormData();
  Array.from(files).forEach((file, index) => {
    const key = `file_${index}`;
    form.append(key, file, file.name);
    form.append(`${key}_path`, file.webkitRelativePath || "");
  });
  const payload = await api("/api/import", { method: "POST", body: form });
  alert(`已导入并解析 ${payload.questionCount} 题。单文件会显示在“导入文件”，目录会按原树形结构显示。`);
  await loadLibrary();
  await loadExamList();
}

