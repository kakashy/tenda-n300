/* tenda-n300(1) — living man page behaviour.
   Command explorer transcripts are the tool's real output format
   (see output.go) rendered on sample data. */

(function () {
  "use strict";

  var prefersReducedMotion =
    window.matchMedia &&
    window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  /* ---------------- command explorer ---------------- */

  var COMMANDS = {
    status: {
      syn: "Print router summary: total, online, blocked.",
      title: "status",
      lines: [
        '<span class="tp">$</span> tenda-n300 status',
        "total devices: <b>9</b>",
        "online:        <b>7</b>",
        "blocked:       <b>2</b>",
      ],
    },
    devices: {
      syn: "List every connected and blocked device.",
      title: "devices",
      lines: [
        '<span class="tp">$</span> tenda-n300 devices',
        "HOSTNAME           IP              MAC               TYPE     ACCESS",
        "living-room-tv     192.168.0.101   A4:53:3C:11:2F:01  TV       <span class=\"tr\">blocked</span>",
        "kitchen-speaker    192.168.0.103   B0:4E:26:5A:8C:77  speaker  allowed",
        "thinkpad           192.168.0.105   3C:52:82:1E:9A:44  laptop   allowed",
        "pixel-8            192.168.0.107   9C:FC:01:32:7B:90  phone    allowed",
        "doorbell           192.168.0.113   00:2E:5F:19:04:6B  camera   allowed",
        "raspi              192.168.0.114   B8:27:EB:63:4A:1F  raspi    allowed",
        "guest-iphone       192.168.0.130   F0:2F:74:3D:8E:C1  phone    <span class=\"tr\">blocked</span>",
      ],
    },
    block: {
      syn: "Block a device by MAC address. Cuts its internet access.",
      title: "block",
      lines: [
        '<span class="tp">$</span> tenda-n300 block F0:2F:74:3D:8E:C1',
        "<span class=\"tr\">blocked F0:2F:74:3D:8E:C1 (guest-iphone)</span>",
      ],
    },
    unblock: {
      syn: "Restore internet access for a blocked MAC address.",
      title: "unblock",
      lines: [
        '<span class="tp">$</span> tenda-n300 unblock F0:2F:74:3D:8E:C1',
        "unblocked F0:2F:74:3D:8E:C1 (guest-iphone)",
      ],
    },
    wifi: {
      syn: "Show current WiFi settings: SSID, password, channel, encryption.",
      title: "wifi",
      lines: [
        '<span class="tp">$</span> tenda-n300 wifi',
        "SSID:       HomeNet",
        "Password:   ********",
        "Channel:    6",
        "Encryption: WPA2PSK/AES",
        "Band:       2.4GHz",
        "WPS:        On",
        "Broadcast:  On",
      ],
    },
    "wifi-set": {
      syn: "Change WiFi settings with any combination of flags.",
      title: "wifi --set",
      lines: [
        '<span class="tp">$</span> tenda-n300 wifi --ssid "HomeNet-5G" --channel 11',
        '<span class="ti">SSID:       HomeNet-5G</span>',
        '<span class="ti">Channel:    11</span>',
        '<span class="ti">Encryption: WPA2PSK/AES</span>',
      ],
    },
    reboot: {
      syn: "Restart the router remotely.",
      title: "reboot",
      lines: [
        '<span class="tp">$</span> tenda-n300 reboot',
        '<span class="ti">rebooting router... (may take ~90 seconds)</span>',
        "done",
      ],
    },
    json: {
      syn: "Any command emits machine-readable JSON with --json.",
      title: "--json status",
      lines: [
        '<span class="tp">$</span> tenda-n300 --json status',
        "{",
        '  "total": 9,',
        '  "online": 7,',
        '  "blocked": 2,',
        '  "devices": [',
        '    { "hostname": "raspi", "ip": "192.168.0.114", "mac": "B8:27:EB:63:4A:1F", "type": "raspi", "access": true },',
        '    { "hostname": "guest-iphone", "ip": "192.168.0.130", "mac": "F0:2F:74:3D:8E:C1", "type": "phone", "access": false }',
        "  ]",
        "}",
      ],
    },
    discover: {
      syn: "Scan the local /24 subnet for Tenda routers.",
      title: "discover",
      lines: [
        '<span class="tp">$</span> tenda-n300 discover',
        "scanning 192.168.0.0/24...",
        "found Tenda router at 192.168.0.1 (N300)",
        "save as profile? [y/N]: y",
        "<span class=\"ti\">profile 'default' saved</span>",
      ],
    },
    profile: {
      syn: "Manage multiple routers: home, work, anywhere.",
      title: "profile",
      lines: [
        '<span class="tp">$</span> tenda-n300 profile',
        "active:  home (192.168.0.1)",
        "default: home",
        '<span class="tp">$</span> tenda-n300 --profile work devices',
        "using profile: work",
      ],
    },
    backup: {
      syn: "Download the router configuration as a backup file.",
      title: "backup",
      lines: [
        '<span class="tp">$</span> tenda-n300 backup',
        '<span class="ti">saved RouterCfm.cfg</span>',
        '<span class="tp">$</span> tenda-n300 restore RouterCfm.cfg',
        '<span class="ti">restored config</span>',
      ],
    },
    syslog: {
      syn: "Export the router's system log to a file or stdout.",
      title: "syslog",
      lines: [
        '<span class="tp">$</span> tenda-n300 syslog /tmp/router.log',
        "wrote 128 lines to /tmp/router.log",
      ],
    },
  };

  function runExplorer() {
    var root = document.querySelector("[data-explorer]");
    if (!root) return;
    var buttons = root.querySelectorAll("[data-command]");
    var syn = root.querySelector("[data-syn]");
    var out = root.querySelector("[data-term-out]");
    var title = root.querySelector("[data-term-title]");

    function render(id) {
      var cmd = COMMANDS[id];
      if (!cmd) return;
      for (var i = 0; i < buttons.length; i++) {
        buttons[i].setAttribute(
          "aria-pressed",
          buttons[i].getAttribute("data-command") === id ? "true" : "false",
        );
      }
      syn.innerHTML = "";
      syn.appendChild(document.createTextNode(""));
      var synCode = document.createElement("b");
      synCode.textContent = id + " ";
      var synText = document.createTextNode("\u2014 " + cmd.syn);
      syn.appendChild(synCode);
      syn.appendChild(synText);
      title.textContent = cmd.title;
      typeLines(out, cmd.lines);
    }

    function typeLines(el, lines) {
      el.innerHTML = "";
      var code = document.createElement("code");
      el.appendChild(code);
      if (prefersReducedMotion) {
        for (var j = 0; j < lines.length; j++) {
          appendLine(code, lines[j]);
        }
        return;
      }
      var i = 0;
      (function next() {
        if (i >= lines.length) return;
        var isLast = i === lines.length - 1;
        appendLine(code, lines[i]);
        i++;
        var delay = isLast ? 0 : 120;
        setTimeout(next, delay);
      })();
    }

    function appendLine(code, html) {
      var line = document.createElement("span");
      line.className = "term__line";
      line.style.display = "block";
      line.innerHTML = html;
      code.appendChild(line);
    }

    buttons.forEach(function (btn) {
      btn.addEventListener("click", function () {
        render(btn.getAttribute("data-command"));
      });
    });

    render("status");
  }

  /* ---------------- copy button ---------------- */

  function runCopy() {
    var btn = document.querySelector("[data-copy]");
    if (!btn) return;
    btn.addEventListener("click", function () {
      var text = btn.getAttribute("data-copy");
      var label = btn.querySelector("[data-copy-label]");
      function done() {
        if (label) label.textContent = "copied";
        btn.classList.add("copy--done");
        setTimeout(function () {
          if (label) label.textContent = "copy";
          btn.classList.remove("copy--done");
        }, 1600);
      }
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(done, function () {
          fallback(text);
          done();
        });
      } else {
        fallback(text);
        done();
      }
    });
    function fallback(text) {
      var ta = document.createElement("textarea");
      ta.value = text;
      ta.setAttribute("readonly", "");
      ta.style.position = "absolute";
      ta.style.left = "-9999px";
      document.body.appendChild(ta);
      ta.select();
      try {
        document.execCommand("copy");
      } catch (e) {
        /* no-op */
      }
      document.body.removeChild(ta);
    }
  }

  /* ---------------- pager + jump ---------------- */

  function runPager() {
    var pager = document.querySelector(".pager");
    var pos = document.querySelector("[data-pos]");
    if (!pager) return;

    var atTop = true;
    function onScroll() {
      var y = window.scrollY;
      var max = document.documentElement.scrollHeight - window.innerHeight;
      var pct = max > 0 ? Math.round((y / max) * 100) : 0;
      if (pos) pos.textContent = pct + "%";
      if (y > 260 && atTop) {
        pager.classList.remove("pager--hidden");
        atTop = false;
      } else if (y <= 260 && !atTop) {
        pager.classList.add("pager--hidden");
        atTop = true;
      }
    }
    window.addEventListener("scroll", onScroll, { passive: true });
    onScroll();

    var jumps = pager.querySelectorAll("[data-jump]");
    jumps.forEach(function (btn) {
      btn.addEventListener("click", function () {
        var id = btn.getAttribute("data-jump");
        var target = document.getElementById(id);
        if (target) target.scrollIntoView({ behavior: "smooth", block: "start" });
      });
    });
  }

  /* ---------------- boot ---------------- */

  document.addEventListener("DOMContentLoaded", function () {
    runExplorer();
    runCopy();
    runPager();
  });
})();
