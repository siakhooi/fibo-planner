(function () {
  const roomID = document.body.dataset.roomId || "";
  const maxDisplayNameLen = Number(document.body.dataset.maxDisplayNameLen);
  const maxPreloadedTopics = Number(document.body.dataset.maxPreloadedTopics) || 200;
  const key = "fibo-planner-display-name-" + roomID;
  const copyRoomUrl = document.getElementById("copy-room-url");
  let copyRoomUrlTimer = 0;
  function markRoomUrlCopied() {
    if (!copyRoomUrl) {
      return;
    }
    copyRoomUrl.classList.add("is-copied");
    copyRoomUrl.setAttribute("aria-label", "Room link copied");
    copyRoomUrl.setAttribute("title", "Copied");
    window.clearTimeout(copyRoomUrlTimer);
    copyRoomUrlTimer = window.setTimeout(function () {
      copyRoomUrl.classList.remove("is-copied");
      copyRoomUrl.setAttribute("aria-label", "Copy room link");
      copyRoomUrl.setAttribute("title", "Copy room link");
    }, 1500);
  }
  function copyRoomLink() {
    const url = window.location.origin + "/" + roomID;
    if (navigator.clipboard?.writeText) {
      navigator.clipboard.writeText(url).then(markRoomUrlCopied).catch(function () {
        fallbackCopyRoomLink(url);
      });
      return;
    }
    fallbackCopyRoomLink(url);
  }
  function fallbackCopyRoomLink(url) {
    const ta = document.createElement("textarea");
    ta.value = url;
    ta.setAttribute("readonly", "");
    ta.style.position = "fixed";
    ta.style.left = "-9999px";
    document.body.appendChild(ta);
    ta.select();
    try {
      if (document.execCommand("copy")) {
        markRoomUrlCopied();
      }
    } finally {
      ta.remove();
    }
  }
  if (copyRoomUrl) {
    copyRoomUrl.addEventListener("click", copyRoomLink);
  }
  const gate = document.getElementById("join-gate");
  const main = document.getElementById("room-main");
  const form = document.getElementById("join-form");
  const input = document.getElementById("display-name");
  const pointsForm = document.getElementById("points-form");
  const pointsValue = document.getElementById("points-value");

  const adminPanel = document.querySelector("details.admin-panel");
  const desktopAdmin = window.matchMedia("(min-width: 60.01rem)");
  function syncAdminPanelOpen() {
    if (!adminPanel) {
      return;
    }
    if (desktopAdmin.matches) {
      adminPanel.setAttribute("open", "");
      delete adminPanel.dataset.userToggled;
      return;
    }
    if (!("userToggled" in adminPanel.dataset)) {
      adminPanel.removeAttribute("open");
    }
  }
  if (adminPanel) {
    adminPanel.addEventListener("toggle", function () {
      if (!desktopAdmin.matches) {
        adminPanel.dataset.userToggled = "";
      }
    });
    if (desktopAdmin.addEventListener) {
      desktopAdmin.addEventListener("change", syncAdminPanelOpen);
    } else if (desktopAdmin.addListener) {
      desktopAdmin.addListener(syncAdminPanelOpen);
    }
    syncAdminPanelOpen();
  }

  function showMyVote() {
    const observerBtn = document.getElementById("observer-mode");
    const voteButtons = pointsForm.querySelectorAll("button");
    const me = document.querySelector("#user-list tbody tr.current-user");
    if (!me) {
      return;
    }
    const cells = me.cells;
    const isObserver = cells[1].textContent === "observer";
    observerBtn.setAttribute("aria-pressed", isObserver ? "true" : "false");
    for (const voteBtn of voteButtons) {
      voteBtn.disabled = isObserver;
    }
    if (isObserver || cells[1].textContent === "") {
      pointsValue.value = "";
      return;
    }
    const mine = pointsValue.value;
    if (mine && cells[1].textContent === "???") {
      cells[1].textContent = mine;
    }
  }

  const roomWs = document.getElementById("room-ws");
  const wsStatus = document.getElementById("ws-status");
  let joinedName = "";
  function setWSDisconnected(disconnected) {
    if (!wsStatus) {
      return;
    }
    wsStatus.hidden = !disconnected;
  }
  if (roomWs) {
    roomWs.addEventListener("htmx:wsAfterMessage", showMyVote);
    roomWs.addEventListener("htmx:wsClose", function () {
      setWSDisconnected(true);
    });
    roomWs.addEventListener("htmx:wsError", function () {
      setWSDisconnected(true);
    });
    roomWs.addEventListener("htmx:wsOpen", function (evt) {
      setWSDisconnected(false);
      const wrapper = evt.detail && evt.detail.socketWrapper;
      if (wrapper && joinedName) {
        wrapper.send(JSON.stringify({ name: joinedName }));
      }
    });
  }

  pointsForm.addEventListener("click", function (e) {
    const btn = e.target.closest("button[data-points]");
    if (!btn) return;
    pointsValue.value = btn.dataset.points;
  });

  function truncateRunes(s, n) {
    return Array.from(s || "").slice(0, n).join("");
  }

  function enterRoom(displayName) {
    displayName = truncateRunes((displayName || "").trim(), maxDisplayNameLen);
    if (!displayName) {
      return;
    }
    localStorage.setItem(key, displayName);
    joinedName = displayName;
    gate.hidden = true;
    main.hidden = false;
    document.getElementById("user-name").textContent = displayName;
    const el = document.getElementById("room-ws");
    el.setAttribute("ws-connect", "/ws/" + roomID);
    document.querySelectorAll(".js-ws-send").forEach(function (formEl) {
      formEl.setAttribute("ws-send", "");
    });
    htmx.process(el);
  }

  const consensusForm = document.getElementById("consensus-form");
  const sliderOutputs = {
    "consensus-percent": "consensus-percent-value",
    "consensus-max-spread": "consensus-max-spread-value"
  };
  consensusForm.addEventListener("input", function (e) {
    const outId = sliderOutputs[e.target.id];
    if (!outId) {
      return;
    }
    const out = document.getElementById(outId);
    if (out) {
      out.textContent = e.target.value;
    }
  });
  consensusForm.addEventListener("change", function (e) {
    if (!sliderOutputs[e.target.id]) {
      return;
    }
    if (consensusForm.getAttribute("ws-send") === null) {
      return;
    }
    consensusForm.requestSubmit();
  });
  consensusForm.addEventListener("click", function (e) {
    const btn = e.target.closest("button.maturity-preset");
    if (!btn) {
      return;
    }
    const percent = btn.dataset.percentage;
    const spread = btn.dataset.maxSpread;
    const percentInput = document.getElementById("consensus-percent");
    const spreadInput = document.getElementById("consensus-max-spread");
    percentInput.value = percent;
    spreadInput.value = spread;
    document.getElementById("consensus-percent-value").textContent = percent;
    document.getElementById("consensus-max-spread-value").textContent = spread;
    if (consensusForm.getAttribute("ws-send") === null) {
      return;
    }
    consensusForm.requestSubmit();
  });

  const preloadedDialog = document.getElementById("preloaded-topics-dialog");
  const preloadedInput = document.getElementById("preloaded-topics-input");
  const preloadedForm = document.getElementById("set-preloaded-topics-form");

  function remainingPreloadedTopics() {
    const data = document.getElementById("preloaded-topics-data");
    return data ? data.textContent : "";
  }

  function fillPreloadedEditor() {
    preloadedInput.value = remainingPreloadedTopics();
  }

  document.getElementById("edit-preloaded-topics").addEventListener("click", function () {
    fillPreloadedEditor();
    if (!preloadedDialog.open) {
      preloadedDialog.showModal();
    }
  });
  document.getElementById("preloaded-topics-cancel").addEventListener("click", function () {
    preloadedDialog.close();
  });
  preloadedForm.addEventListener("submit", function () {
    const lines = preloadedInput.value.split(/\r?\n/);
    const kept = [];
    for (const raw of lines) {
      const line = raw.trim();
      if (!line) {
        continue;
      }
      if (kept.length >= maxPreloadedTopics) {
        break;
      }
      kept.push(truncateRunes(line, maxDisplayNameLen));
    }
    preloadedInput.value = kept.join("\n");
    const data = document.getElementById("preloaded-topics-data");
    if (data) {
      data.textContent = preloadedInput.value;
    }
    preloadedDialog.close();
  });

  const stored = localStorage.getItem(key);
  if (stored && stored.trim()) {
    input.value = stored.trim();
  }
  input.focus();
  form.addEventListener("submit", function (e) {
    e.preventDefault();
    enterRoom(input.value);
  });
})();
