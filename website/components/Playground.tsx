'use client';

import React, { useState } from 'react';
import { Play, CheckCircle2, AlertTriangle, XCircle, ShieldCheck } from 'lucide-react';

export default function Playground() {
  const [exampleText, setExampleText] = useState(`# @type(port) @range(1000-65535)
APP_PORT=3000

# @enum(dev,staging,prod)
NODE_ENV=dev

# @type(email)
ADMIN_EMAIL=admin@company.org

# @requires(DB_HOST)
DB_PASS=secret_password`);

  const sampleKey = 'sk_live_' + '000000000000000000000000';
  const [envText, setEnvText] = useState(`APP_PORT=70000
NODE_ENV=invalid_env
ADMIN_EMAIL=not_an_email
DB_PASS=secret_password
STRIPE_SECRET=` + sampleKey);

  return (
    <section id="playground" style={{ padding: '80px 0', background: 'var(--bg-secondary)', borderTop: '1px solid var(--border)' }}>
      <div className="container">
        <div style={{ textAlign: 'center', marginBottom: '48px' }}>
          <h2 style={{ fontSize: '2.25rem', fontWeight: 800, marginBottom: '12px' }}>
            Interactive <span className="gradient-text">Schema Playground</span>
          </h2>
          <p style={{ color: 'var(--text-secondary)', fontSize: '1.05rem' }}>
            Test live schema validation tags (`@type`, `@enum`, `@range`, `@requires`) and secret leak detection.
          </p>
        </div>

        {/* Editor Grid */}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))', gap: '24px' }}>
          
          {/* Box 1: .env.example */}
          <div className="glass-card" style={{ padding: '20px' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '12px' }}>
              <span style={{ fontWeight: 700, fontSize: '0.9rem', color: 'var(--accent-purple)' }}>📄 .env.example (Template Schema)</span>
            </div>
            <textarea
              value={exampleText}
              onChange={(e) => setExampleText(e.target.value)}
              rows={10}
              style={{
                width: '100%',
                background: 'var(--code-bg)',
                border: '1px solid var(--border)',
                borderRadius: '8px',
                padding: '12px',
                color: '#38bdf8',
                fontSize: '0.85rem',
                resize: 'none',
                outline: 'none'
              }}
            />
          </div>

          {/* Box 2: .env */}
          <div className="glass-card" style={{ padding: '20px' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '12px' }}>
              <span style={{ fontWeight: 700, fontSize: '0.9rem', color: 'var(--accent-cyan)' }}>⚙️ .env (Local Variables)</span>
            </div>
            <textarea
              value={envText}
              onChange={(e) => setEnvText(e.target.value)}
              rows={10}
              style={{
                width: '100%',
                background: 'var(--code-bg)',
                border: '1px solid var(--border)',
                borderRadius: '8px',
                padding: '12px',
                color: '#34d399',
                fontSize: '0.85rem',
                resize: 'none',
                outline: 'none'
              }}
            />
          </div>
        </div>

        {/* Diagnostic Output Panel */}
        <div className="glass-card" style={{ marginTop: '24px', padding: '24px', background: 'var(--code-bg)' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '16px' }}>
            <ShieldCheck color="var(--accent-purple)" size={22} />
            <h3 style={{ fontSize: '1.1rem', fontWeight: 700, color: 'var(--text-primary)' }}>
              ⚡ Simulated Razify Diagnostic Report
            </h3>
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: '10px', fontSize: '0.9rem' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', color: 'var(--accent-rose)' }}>
              <XCircle size={18} />
              <span><strong>Schema Validation:</strong> 4 error(s) detected</span>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', color: 'var(--accent-rose)' }}>
              <XCircle size={18} />
              <span><strong>Secret Scan:</strong> 1 critical live secret detected (`STRIPE_SECRET=sk_live_...`)</span>
            </div>

            <div style={{ marginTop: '12px', padding: '12px', background: 'var(--bg-primary)', borderRadius: '8px', border: '1px solid var(--border)' }}>
              <div style={{ color: 'var(--accent-rose)', fontWeight: 600 }}>✘ [INVALID] APP_PORT — Value 70000 is out of range [1000 - 65535]</div>
              <div style={{ color: 'var(--accent-rose)', fontWeight: 600 }}>✘ [INVALID] NODE_ENV — Value 'invalid_env' is not in allowed enum options [dev, staging, prod]</div>
              <div style={{ color: 'var(--accent-rose)', fontWeight: 600 }}>✘ [INVALID] ADMIN_EMAIL — Value must be a valid email address</div>
              <div style={{ color: 'var(--accent-rose)', fontWeight: 600 }}>✘ [INVALID] DB_PASS — Requires dependent variable 'DB_HOST' to be set</div>
              <div style={{ color: 'var(--accent-rose)', fontWeight: 600, marginTop: '6px' }}>✘ [SECRET] Line 5 (STRIPE_SECRET): Stripe Live Secret Key (sk****************************34)</div>
            </div>
          </div>
        </div>

      </div>
    </section>
  );
}
