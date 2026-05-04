import React from "react";

type Props = {
  count: number;
  notice?: string;
  error?: string;
};

export function App({ count, notice, error }: Props) {
  return (
    <html lang="en">
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <title>Visit Counter</title>
        <style>{`
          :root { --bg: #f2efe8; --panel: #fffaf2; --text: #1f2937; --accent: #0f766e; }
          body { margin: 0; font-family: "IBM Plex Sans", "Segoe UI", sans-serif; color: var(--text); background: radial-gradient(circle at top right, #dbeafe, var(--bg)); min-height: 100vh; display: grid; place-items: center; }
          .card { background: var(--panel); border: 1px solid #d1d5db; border-radius: 14px; width: min(92vw, 560px); padding: 28px; box-shadow: 0 20px 50px rgba(15, 23, 42, 0.08); }
          h1 { margin: 0 0 8px; }
          p { margin: 0 0 18px; }
          button { background: var(--accent); color: #fff; border: 0; border-radius: 10px; padding: 12px 16px; font-weight: 600; cursor: pointer; }
          .row { display: flex; gap: 12px; align-items: center; }
          .count { font-size: 1.75rem; font-weight: 700; }
          .msg { min-height: 24px; margin-top: 16px; }
          .ok { color: #166534; }
          .err { color: #991b1b; }
        `}</style>
      </head>
      <body>
        <main className="card">
          <h1>Visit Counter</h1>
          <p>Server-rendered count. Click queues a visit and re-renders with fresh data.</p>
          <div className="row">
            <form method="post" action="/visit">
              <button type="submit">Register Visit</button>
            </form>
            <span>
              Total visits: <strong className="count">{count}</strong>
            </span>
          </div>
          <p className={`msg ${error ? "err" : "ok"}`}>{error ?? notice ?? ""}</p>
        </main>
      </body>
    </html>
  );
}
