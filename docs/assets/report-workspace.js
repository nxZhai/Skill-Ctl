(function () {
  var STORAGE_PREFIX = "organize-html-reports:";
  var annotationRange = null;
  var activeAnnotation = null;

  function showToast(message) {
    var toast = document.querySelector(".copy-toast");
    if (!toast) {
      toast = document.createElement("div");
      toast.className = "copy-toast";
      toast.setAttribute("role", "status");
      toast.setAttribute("aria-live", "polite");
      document.body.appendChild(toast);
    }
    toast.textContent = message;
    toast.classList.add("visible");
    window.clearTimeout(showToast.timer);
    showToast.timer = window.setTimeout(function () {
      toast.classList.remove("visible");
    }, 1600);
  }

  function fallbackCopy(text) {
    var area = document.createElement("textarea");
    area.value = text;
    area.setAttribute("readonly", "");
    area.style.position = "fixed";
    area.style.left = "-9999px";
    document.body.appendChild(area);
    area.select();
    try {
      document.execCommand("copy");
    } finally {
      document.body.removeChild(area);
    }
  }

  function copyText(text, message) {
    if (!text) {
      return;
    }
    if (navigator.clipboard && window.isSecureContext) {
      navigator.clipboard.writeText(text).then(
        function () {
          showToast(message);
        },
        function () {
          fallbackCopy(text);
          showToast(message);
        }
      );
      return;
    }
    fallbackCopy(text);
    showToast(message);
  }

  function storageKey(kind, id) {
    return STORAGE_PREFIX + kind + ":" + id;
  }

  function pageNode() {
    return document.querySelector(".report-article") || document.querySelector("[data-page-role]");
  }

  function pageId() {
    var node = pageNode();
    return (node && node.dataset.pageId) || window.location.pathname;
  }

  function pageRole() {
    var node = pageNode();
    return (node && node.dataset.pageRole) || "subpage";
  }

  function articleBody() {
    return document.querySelector(".article-body") || pageNode();
  }

  function absoluteHref(href) {
    try {
      return new URL(href, window.location.href).href;
    } catch (error) {
      return href;
    }
  }

  var MATERIAL_SYMBOLS_VIEWBOX = "0 -960 960 960";
  var ICONS = {
    arrow_back: {
      viewBox: MATERIAL_SYMBOLS_VIEWBOX,
      path: "m313-440 224 224-57 56-320-320 320-320 57 56-224 224h487v80H313Z",
    },
    content_copy: {
      viewBox: MATERIAL_SYMBOLS_VIEWBOX,
      path: "M360-240q-33 0-56.5-23.5T280-320v-480q0-33 23.5-56.5T360-880h360q33 0 56.5 23.5T800-800v480q0 33-23.5 56.5T720-240H360Zm0-80h360v-480H360v480ZM200-80q-33 0-56.5-23.5T120-160v-560h80v560h440v80H200Zm160-240v-480 480Z",
    },
    dark_mode: {
      viewBox: MATERIAL_SYMBOLS_VIEWBOX,
      path: "M480-120q-150 0-255-105T120-480q0-150 105-255t255-105q14 0 27.5 1t26.5 3q-41 29-65.5 75.5T444-660q0 90 63 153t153 63q55 0 101-24.5t75-65.5q2 13 3 26.5t1 27.5q0 150-105 255T480-120Zm0-80q88 0 158-48.5T740-375q-20 5-40 8t-40 3q-123 0-209.5-86.5T364-660q0-20 3-40t8-40q-78 32-126.5 102T200-480q0 116 82 198t198 82Zm-10-270Z",
    },
    delete: {
      viewBox: MATERIAL_SYMBOLS_VIEWBOX,
      path: "M280-120q-33 0-56.5-23.5T200-200v-520h-40v-80h200v-40h240v40h200v80h-40v520q0 33-23.5 56.5T680-120H280Zm400-600H280v520h400v-520ZM360-280h80v-360h-80v360Zm160 0h80v-360h-80v360ZM280-720v520-520Z",
    },
    folder_open: {
      viewBox: MATERIAL_SYMBOLS_VIEWBOX,
      path: "M160-160q-33 0-56.5-23.5T80-240v-480q0-33 23.5-56.5T160-800h240l80 80h320q33 0 56.5 23.5T880-640H447l-80-80H160v480l96-320h684L837-217q-8 26-29.5 41.5T760-160H160Zm84-80h516l72-240H316l-72 240Zm0 0 72-240-72 240Zm-84-400v-80 80Z",
    },
    format_underlined: {
      viewBox: MATERIAL_SYMBOLS_VIEWBOX,
      path: "M200-120v-80h560v80H200Zm123-223q-56-63-56-167v-330h103v336q0 56 28 91t82 35q54 0 82-35t28-91v-336h103v330q0 104-56 167t-157 63q-101 0-157-63Z",
    },
    home: {
      viewBox: MATERIAL_SYMBOLS_VIEWBOX,
      path: "M240-200h120v-240h240v240h120v-360L480-740 240-560v360Zm-80 80v-480l320-240 320 240v480H520v-240h-80v240H160Zm320-350Z",
    },
    light_mode: {
      viewBox: MATERIAL_SYMBOLS_VIEWBOX,
      path: "M565-395q35-35 35-85t-35-85q-35-35-85-35t-85 35q-35 35-35 85t35 85q35 35 85 35t85-35Zm-226.5 56.5Q280-397 280-480t58.5-141.5Q397-680 480-680t141.5 58.5Q680-563 680-480t-58.5 141.5Q563-280 480-280t-141.5-58.5ZM200-440H40v-80h160v80Zm720 0H760v-80h160v80ZM440-760v-160h80v160h-80Zm0 720v-160h80v160h-80ZM256-650l-101-97 57-59 96 100-52 56Zm492 496-97-101 53-55 101 97-57 59Zm-98-550 97-101 59 57-100 96-56-52ZM154-212l101-97 55 53-97 101-59-57Zm326-268Z",
    },
    open_in_new: {
      viewBox: MATERIAL_SYMBOLS_VIEWBOX,
      path: "M200-120q-33 0-56.5-23.5T120-200v-560q0-33 23.5-56.5T200-840h280v80H200v560h560v-280h80v280q0 33-23.5 56.5T760-120H200Zm188-212-56-56 372-372H560v-80h280v280h-80v-144L388-332Z",
    },
    palette: {
      viewBox: MATERIAL_SYMBOLS_VIEWBOX,
      path: "M480-80q-82 0-155-31.5t-127.5-86Q143-252 111.5-325T80-480q0-83 32.5-156t88-127Q256-817 330-848.5T488-880q80 0 151 27.5t124.5 76q53.5 48.5 85 115T880-518q0 115-70 176.5T640-280h-74q-9 0-12.5 5t-3.5 11q0 12 15 34.5t15 51.5q0 50-27.5 74T480-80Zm0-400Zm-177 23q17-17 17-43t-17-43q-17-17-43-17t-43 17q-17 17-17 43t17 43q17 17 43 17t43-17Zm120-160q17-17 17-43t-17-43q-17-17-43-17t-43 17q-17 17-17 43t17 43q17 17 43 17t43-17Zm200 0q17-17 17-43t-17-43q-17-17-43-17t-43 17q-17 17-17 43t17 43q17 17 43 17t43-17Zm120 160q17-17 17-43t-17-43q-17-17-43-17t-43 17q-17 17-17 43t17 43q17 17 43 17t43-17ZM480-160q9 0 14.5-5t5.5-13q0-14-15-33t-15-57q0-42 29-67t71-25h70q66 0 113-38.5T800-518q0-121-92.5-201.5T488-800q-136 0-232 93t-96 227q0 133 93.5 226.5T480-160Z",
    },
  };

  function materialIcon(name) {
    var icon = ICONS[name] || ICONS.open_in_new;
    return (
      '<span class="material-action-icon" aria-hidden="true">' +
      '<svg viewBox="' +
      icon.viewBox +
      '" focusable="false"><path d="' +
      icon.path +
      '"></path></svg></span>'
    );
  }

  function setIconButton(node, icon, label) {
    if (!node) {
      return;
    }
    node.classList.add("icon-button");
    node.setAttribute("aria-label", label);
    node.title = label;
    node.innerHTML = materialIcon(icon);
  }

  function cellText(cell) {
    return cell.textContent.replace(/\s+/g, " ").trim().replace(/\|/g, "\\|");
  }

  function tableToMarkdown(table) {
    var rows = Array.prototype.slice.call(table.rows).map(function (row) {
      return Array.prototype.slice.call(row.cells)
        .filter(function (cell) {
          return !cell.dataset.tableActionCell;
        })
        .map(cellText);
    });
    if (!rows.length) {
      return "";
    }
    var width = rows.reduce(function (max, row) {
      return Math.max(max, row.length);
    }, 0);
    rows = rows.map(function (row) {
      while (row.length < width) {
        row.push("");
      }
      return row;
    });
    var divider = rows[0].map(function () {
      return "---";
    });
    return [rows[0], divider]
      .concat(rows.slice(1))
      .map(function (row) {
        return "| " + row.join(" | ") + " |";
      })
      .join("\n");
  }

  function enhanceTables() {
    Array.prototype.slice.call(document.querySelectorAll("table")).forEach(function (table) {
      if (table.closest(".table-block")) {
        return;
      }

      var block = document.createElement("div");
      block.className = "table-block";
      var toolbar = document.createElement("div");
      toolbar.className = "table-toolbar";
      var frame = document.createElement("div");
      frame.className = "table-frame";

      table.parentNode.insertBefore(block, table);
      block.appendChild(toolbar);
      block.appendChild(frame);
      frame.appendChild(table);

      var button = document.createElement("button");
      button.type = "button";
      button.className = "table-copy";
      setIconButton(button, "content_copy", "Copy table as Markdown");
      button.addEventListener("click", function () {
        copyText(table.dataset.markdownSource || tableToMarkdown(table), "Markdown table copied");
      });
      if (!installTableActionColumn(table, button)) {
        toolbar.appendChild(button);
      }
    });
  }

  function installTableActionColumn(table, button) {
    var rows = Array.prototype.slice.call(table.rows);
    if (!rows.length) {
      return false;
    }

    var headerRow = table.tHead && table.tHead.rows.length ? table.tHead.rows[0] : rows[0];
    var headerCellTag = Array.prototype.some.call(headerRow.cells, function (cell) {
      return cell.tagName && cell.tagName.toLowerCase() === "th";
    })
      ? "th"
      : "td";
    var headerCell = document.createElement(headerCellTag);
    headerCell.className = "table-action-cell";
    headerCell.dataset.tableActionCell = "true";
    headerCell.setAttribute("aria-label", "Table actions");
    if (headerCellTag === "th") {
      headerCell.setAttribute("scope", "col");
    }
    headerCell.appendChild(button);
    headerRow.appendChild(headerCell);

    rows.forEach(function (row) {
      if (row === headerRow) {
        return;
      }
      var cell = document.createElement("td");
      cell.className = "table-action-cell";
      cell.dataset.tableActionCell = "true";
      cell.setAttribute("aria-hidden", "true");
      row.appendChild(cell);
    });
    return true;
  }

  function normalizeInlineMath() {
    var changed = [];
    Array.prototype.slice.call(document.querySelectorAll(".math-inline[data-latex]")).forEach(function (node) {
      if (node.querySelector("mjx-container, svg")) {
        return;
      }
      var latex = node.dataset.latex || "";
      if (!latex) {
        return;
      }
      var text = node.textContent.trim();
      if (text.slice(0, 2) === "\\(" && text.slice(-2) === "\\)") {
        return;
      }
      node.textContent = "\\(" + latex + "\\)";
      changed.push(node);
    });
    if (changed.length && window.MathJax && window.MathJax.typesetPromise) {
      window.MathJax.typesetPromise(changed).catch(function () {});
    }
  }

  function enhanceMath() {
    document.addEventListener("click", function (event) {
      var target = event.target.closest && event.target.closest(".math-copy");
      if (target) {
        copyText(target.dataset.latex, "LaTeX copied");
      }
    });
    document.addEventListener("keydown", function (event) {
      if (event.key !== "Enter" && event.key !== " ") {
        return;
      }
      var target = event.target.closest && event.target.closest(".math-copy");
      if (target) {
        event.preventDefault();
        copyText(target.dataset.latex, "LaTeX copied");
      }
    });
  }

  function controlText(node) {
    return (node.textContent || "").replace(/\s+/g, " ").trim().toLowerCase();
  }

  function enhanceThemeToggle(button) {
    if (!button) {
      return;
    }
    function apply() {
      var text = controlText(button);
      var isDarkAction = text.indexOf("dark") !== -1 || text.indexOf("深色") !== -1;
      setIconButton(button, isDarkAction ? "dark_mode" : "light_mode", isDarkAction ? "Dark theme" : "Light theme");
    }
    apply();
    button.addEventListener("click", function () {
      window.setTimeout(apply, 0);
    });
  }

  function enhanceMaterialControls() {
    Array.prototype.slice.call(document.querySelectorAll(".desktop-index-link")).forEach(function (node) {
      setIconButton(node, "home", "Desktop Index");
    });
    Array.prototype.slice.call(document.querySelectorAll(".project-index-link")).forEach(function (node) {
      setIconButton(node, "arrow_back", "Project Index");
    });
    Array.prototype.slice.call(document.querySelectorAll('a.open-link, a[aria-label="Open"], .reportActions a:first-child')).forEach(function (node) {
      if (controlText(node) === "open" || controlText(node) === "打开 html") {
        setIconButton(node, "open_in_new", "Open");
      }
    });
    Array.prototype.slice.call(document.querySelectorAll("a.link-button")).forEach(function (node) {
      if (controlText(node) === "open") {
        setIconButton(node, "open_in_new", "Open");
      }
    });
    Array.prototype.slice.call(document.querySelectorAll(".table-copy")).forEach(function (node) {
      setIconButton(node, "content_copy", "Copy table as Markdown");
    });
    enhanceThemeToggle(document.getElementById("themeToggle"));
  }

  function focusKey() {
    var node = pageNode();
    var source = (node && (node.dataset.workspaceId || node.dataset.sourcePath || node.dataset.pageId)) || window.location.pathname;
    return storageKey("focus", source.replace(/\/docs\/index\.md$|\/index\.md$/, ""));
  }

  function applyWorkspaceFocus() {
    var target = document.querySelector(".workspace-focus-target") || pageNode();
    var active = window.localStorage.getItem(focusKey()) === "1";
    if (target) {
      target.classList.toggle("workspace-focused", active);
    }
    var button = document.querySelector("[data-workspace-focus]");
    if (button) {
      button.textContent = active ? "Focused" : "Focus";
      button.setAttribute("aria-pressed", active ? "true" : "false");
    }
  }

  function enhanceWorkspaceFocus() {
    if (pageRole() !== "project-index") {
      return;
    }
    var nav = document.querySelector(".report-nav") || document.querySelector("[data-workspace-focus-slot]");
    if (!nav) {
      return;
    }
    var button = document.createElement("button");
    button.type = "button";
    button.className = "report-action";
    button.dataset.workspaceFocus = "true";
    button.addEventListener("click", function () {
      var key = focusKey();
      var next = window.localStorage.getItem(key) === "1" ? "0" : "1";
      window.localStorage.setItem(key, next);
      applyWorkspaceFocus();
    });
    nav.appendChild(button);
    applyWorkspaceFocus();
  }

  function renderIndexNotes() {
    if (pageRole() !== "project-index") {
      return;
    }
    var article = pageNode();
    var header = article && (article.querySelector(".article-header") || article.querySelector("[data-index-notes-anchor]"));
    if (!article) {
      return;
    }
    var links = Array.prototype.slice.call(article.querySelectorAll('a[href$=".html"]')).filter(function (link) {
      var href = absoluteHref(link.getAttribute("href"));
      return href !== window.location.href && href.indexOf("/Desktop/index.html") === -1;
    });
    if (!links.length) {
      return;
    }

    var seen = {};
    links = links.filter(function (link) {
      var href = absoluteHref(link.getAttribute("href"));
      if (seen[href]) {
        return false;
      }
      seen[href] = true;
      return true;
    });

    var panel = document.createElement("section");
    panel.className = "index-notes-panel";
    panel.setAttribute("aria-label", "Page notes");
    panel.innerHTML = '<h2>Page Notes</h2><div class="index-note-list"></div>';
    var list = panel.querySelector(".index-note-list");
    links.forEach(function (link) {
      var href = absoluteHref(link.getAttribute("href"));
      var key = storageKey("note", href);
      var row = document.createElement("label");
      row.className = "index-note-row";
      row.innerHTML =
        '<span class="index-note-title"><span>' +
        link.textContent.trim() +
        '</span><span class="mono">' +
        (link.getAttribute("href") || "") +
        '</span></span><textarea placeholder="Add a note for this page"></textarea>';
      var textarea = row.querySelector("textarea");
      textarea.value = window.localStorage.getItem(key) || "";
      textarea.addEventListener("input", function () {
        window.localStorage.setItem(key, textarea.value);
      });
      list.appendChild(row);
    });
    if (header) {
      header.insertAdjacentElement("afterend", panel);
    } else {
      article.insertBefore(panel, article.firstChild);
    }
  }

  function renderCurrentPageNote() {
    if (pageRole() === "project-index") {
      return;
    }
    var article = pageNode();
    var header = article && (article.querySelector(".article-header") || article.querySelector("[data-index-notes-anchor]"));
    if (!article) {
      return;
    }
    var note = window.localStorage.getItem(storageKey("note", window.location.href));
    if (!note) {
      return;
    }
    var box = document.createElement("div");
    box.className = "current-page-note";
    box.textContent = note;
    if (header) {
      header.insertAdjacentElement("afterend", box);
    } else {
      article.insertBefore(box, article.firstChild);
    }
  }

  function saveAnnotations() {
    var body = articleBody();
    if (body) {
      window.localStorage.setItem(storageKey("annotations", pageId()), body.innerHTML);
    }
  }

  function restoreAnnotations() {
    var body = articleBody();
    var saved = window.localStorage.getItem(storageKey("annotations", pageId()));
    if (body && saved) {
      body.innerHTML = saved;
    }
  }

  function toolbar() {
    var node = document.querySelector(".annotation-toolbar");
    if (node) {
      return node;
    }
    node = document.createElement("div");
    node.className = "annotation-toolbar";
    node.setAttribute("role", "toolbar");
    node.innerHTML = [
      '<button type="button" data-annotation-style="yellow" aria-label="Yellow highlight" title="Yellow highlight"><span class="annotation-swatch annotation-swatch-yellow" aria-hidden="true"></span></button>',
      '<button type="button" data-annotation-style="green" aria-label="Green highlight" title="Green highlight"><span class="annotation-swatch annotation-swatch-green" aria-hidden="true"></span></button>',
      '<button type="button" data-annotation-style="purple" aria-label="Purple highlight" title="Purple highlight"><span class="annotation-swatch annotation-swatch-purple" aria-hidden="true"></span></button>',
      '<button type="button" data-annotation-style="underline" aria-label="Underline" title="Underline">' +
        materialIcon("format_underlined") +
      '</button>',
      '<button type="button" data-annotation-style="delete" aria-label="Delete annotation" title="Delete annotation">' +
        materialIcon("delete") +
      '</button>',
    ].join("");
    node.addEventListener("click", function (event) {
      var button = event.target.closest && event.target.closest("[data-annotation-style]");
      if (!button) {
        return;
      }
      applyAnnotation(button.dataset.annotationStyle);
    });
    document.body.appendChild(node);
    return node;
  }

  function placeToolbar(rect) {
    var node = toolbar();
    var top = Math.max(8, rect.top - 44);
    var left = Math.max(8, Math.min(rect.left, window.innerWidth - 320));
    node.style.top = top + "px";
    node.style.left = left + "px";
    node.classList.add("visible");
  }

  function hideToolbar() {
    var node = toolbar();
    node.classList.remove("visible");
    annotationRange = null;
    activeAnnotation = null;
  }

  function setAnnotationStyle(node, style) {
    node.className = "annotation-mark";
    if (style === "underline") {
      node.classList.add("annotation-underline");
    } else {
      node.classList.add("annotation-" + style);
    }
    node.dataset.annotationStyle = style;
  }

  function unwrap(node) {
    var parent = node.parentNode;
    while (node.firstChild) {
      parent.insertBefore(node.firstChild, node);
    }
    parent.removeChild(node);
    parent.normalize();
  }

  function applyAnnotation(style) {
    if (activeAnnotation) {
      if (style === "delete") {
        unwrap(activeAnnotation);
      } else {
        setAnnotationStyle(activeAnnotation, style);
      }
      saveAnnotations();
      hideToolbar();
      return;
    }
    if (!annotationRange || style === "delete") {
      hideToolbar();
      return;
    }
    var mark = document.createElement("span");
    mark.dataset.annotation = "true";
    setAnnotationStyle(mark, style);
    try {
      mark.appendChild(annotationRange.extractContents());
      annotationRange.insertNode(mark);
    } catch (error) {
      hideToolbar();
      return;
    }
    window.getSelection().removeAllRanges();
    saveAnnotations();
    hideToolbar();
  }

  function enhanceAnnotations() {
    restoreAnnotations();
    document.addEventListener("mouseup", function (event) {
      if (event.target.closest && event.target.closest(".annotation-toolbar, textarea, button")) {
        return;
      }
      var body = articleBody();
      var selection = window.getSelection();
      if (!body || !selection || selection.isCollapsed || !selection.rangeCount) {
        return;
      }
      var range = selection.getRangeAt(0);
      if (!body.contains(range.commonAncestorContainer)) {
        return;
      }
      annotationRange = range.cloneRange();
      activeAnnotation = null;
      placeToolbar(range.getBoundingClientRect());
    });
    document.addEventListener("click", function (event) {
      var mark = event.target.closest && event.target.closest(".annotation-mark");
      if (mark) {
        event.preventDefault();
        activeAnnotation = mark;
        annotationRange = null;
        placeToolbar(mark.getBoundingClientRect());
        return;
      }
      if (event.target.closest && !event.target.closest(".annotation-toolbar")) {
        var selection = window.getSelection();
        if (!selection || selection.isCollapsed) {
          hideToolbar();
        }
      }
    });
  }

  document.addEventListener("DOMContentLoaded", function () {
    enhanceAnnotations();
    enhanceTables();
    normalizeInlineMath();
    enhanceMath();
    enhanceWorkspaceFocus();
    renderIndexNotes();
    renderCurrentPageNote();
    enhanceMaterialControls();
  });
})();
