'use client';

import React from 'react';
import { Zap, Shield, FileCheck, Terminal, Code, GitPullRequest } from 'lucide-react';

const features = [
  {
    icon: <Zap color="var(--accent-purple)" size={28} />,
    title: "Sub-10ms Performance",
    description: "Compiled native Go binary engine. Zero startup interpreter latency, fast enough for every git commit and build loop."
  },
  {
    icon: <Shield color="var(--accent-cyan)" size={28} />,
    title: "Secret Leak Scanning",
    description: "High-entropy analysis and pattern matchers to block exposed API keys, passwords, and JWTs before they reach version control."
  },
  {
    icon: <FileCheck color="var(--accent-emerald)" size={28} />,
    title: "Rich Schema Annotations",
    description: "Enforce @type(email|port|uuid|json), @enum(dev,prod), @range(...), and @requires(...) dependencies in .env.example."
  },
  {
    icon: <Terminal color="var(--accent-rose)" size={28} />,
    title: "Zero-Install npx Runner",
    description: "Run 'npx razify check' anywhere Node.js is installed. Zero global binary setup or installation overhead required."
  },
  {
    icon: <Code color="var(--accent-purple)" size={28} />,
    title: "Official VS Code Extension",
    description: "Real-time inline red/yellow error squigglies, schema hover tooltips, autocompletions, and check-on-save diagnostics."
  },
  {
    icon: <GitPullRequest color="var(--accent-cyan)" size={28} />,
    title: "Git Hooks & GitHub Action",
    description: "Sub-10ms pre-commit git hook ('razify guard install') and 1-step GitHub Action ('uses: Hossiy21/razify@v1') for CI gates."
  }
];

export default function Features() {
  return (
    <section id="features" style={{ padding: '80px 0' }}>
      <div className="container">
        <div style={{ textAlign: 'center', marginBottom: '56px' }}>
          <h2 style={{ fontSize: '2.5rem', fontWeight: 800, marginBottom: '12px' }}>
            Built for <span className="gradient-text">Developer Happiness</span>
          </h2>
          <p style={{ color: 'var(--text-secondary)', fontSize: '1.1rem', maxWidth: '600px', margin: '0 auto' }}>
            Everything you need to make environment variables deterministic, typed, and secure.
          </p>
        </div>

        <div style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))',
          gap: '24px'
        }}>
          {features.map((item, idx) => (
            <div key={idx} className="glass-card" style={{ padding: '28px' }}>
              <div style={{ marginBottom: '16px' }}>{item.icon}</div>
              <h3 style={{ fontSize: '1.2rem', fontWeight: 700, marginBottom: '8px' }}>{item.title}</h3>
              <p style={{ color: 'var(--text-secondary)', fontSize: '0.95rem', lineHeight: 1.6 }}>{item.description}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
