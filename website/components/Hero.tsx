'use client';

import React, { useState } from 'react';
import { Copy, Check, Terminal, Shield, Zap, Sparkles, ArrowRight } from 'lucide-react';

export default function Hero() {
  const [copied, setCopied] = useState(false);
  const command = 'npx razify check';

  const copyCommand = () => {
    navigator.clipboard.writeText(command);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <section style={{ padding: '80px 0 60px 0', textAlign: 'center', position: 'relative' }}>
      <div className="container">
        {/* Badge */}
        <div style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: '8px',
          padding: '6px 16px',
          borderRadius: '20px',
          background: 'var(--glow-purple)',
          border: '1px solid var(--border-glow)',
          color: 'var(--accent-purple)',
          fontWeight: 600,
          fontSize: '0.85rem',
          marginBottom: '24px'
        }}>
          <Sparkles size={16} />
          Razify v1.0.0 is Live on npm
        </div>

        {/* Main Title */}
        <h1 style={{
          fontSize: '3.5rem',
          fontWeight: 800,
          letterSpacing: '-0.03em',
          lineHeight: 1.1,
          marginBottom: '20px'
        }}>
          The <span className="gradient-text">Configuration Integrity Engine</span>
          <br />for Environment Variables.
        </h1>

        {/* Subtitle */}
        <p style={{
          fontSize: '1.2rem',
          color: 'var(--text-secondary)',
          maxWidth: '700px',
          margin: '0 auto 36px auto',
          fontWeight: 400
        }}>
          Type-safe validation, schema enforcement, secret leak scanning, and instant IDE diagnostics. Sub-10ms speed. Zero cloud dependency.
        </p>

        {/* Terminal Copy Snippet */}
        <div style={{
          maxWidth: '540px',
          margin: '0 auto 40px auto',
          background: 'var(--code-bg)',
          borderRadius: '14px',
          border: '1px solid var(--border)',
          padding: '16px 20px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          boxShadow: '0 20px 40px -10px rgba(0,0,0,0.3)'
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <Terminal size={20} color="var(--accent-purple)" />
            <code style={{ fontSize: '1rem', color: '#38bdf8', fontWeight: 600 }}>
              $ npx razify check
            </code>
          </div>
          <button
            onClick={copyCommand}
            style={{
              background: 'transparent',
              border: 'none',
              cursor: 'pointer',
              color: copied ? 'var(--accent-emerald)' : 'var(--text-secondary)',
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
              fontSize: '0.85rem',
              fontWeight: 600
            }}
          >
            {copied ? <Check size={16} /> : <Copy size={16} />}
            {copied ? 'Copied!' : 'Copy'}
          </button>
        </div>

        {/* Action Buttons */}
        <div style={{ display: 'flex', justifyContent: 'center', gap: '16px', flexWrap: 'wrap' }}>
          <a href="#playground" className="btn-primary">
            Try Live Playground <ArrowRight size={18} />
          </a>
          <a href="https://marketplace.visualstudio.com" target="_blank" rel="noreferrer" className="btn-secondary">
            Get VS Code Extension
          </a>
        </div>
      </div>
    </section>
  );
}
