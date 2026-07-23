'use client';

import React, { useState } from 'react';
import { Copy, Check, Terminal, Zap, Shield, Sparkles, ArrowRight, Lock, Cpu } from 'lucide-react';

export default function Hero() {
  const [copied, setCopied] = useState(false);
  const command = 'npx razify check';

  const copyCommand = () => {
    navigator.clipboard.writeText(command);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <section style={{ padding: '90px 0 70px 0', textAlign: 'center', position: 'relative' }}>
      <div className="container">
        
        {/* Release Pill Badge */}
        <div style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: '8px',
          padding: '8px 20px',
          borderRadius: '30px',
          background: 'var(--glow-purple)',
          border: '1px solid var(--border-glow)',
          color: 'var(--accent-purple)',
          fontWeight: 700,
          fontSize: '0.88rem',
          marginBottom: '28px'
        }}>
          <Sparkles size={16} />
          Razify v1.0.1 is Live on npm Registry
        </div>

        {/* Main Headline */}
        <h1 style={{
          fontSize: '3.75rem',
          fontWeight: 800,
          letterSpacing: '-0.035em',
          lineHeight: 1.1,
          marginBottom: '24px'
        }}>
          The <span className="gradient-text">Configuration Integrity Engine</span>
          <br />for Environment Variables.
        </h1>

        {/* Subtitle */}
        <p style={{
          fontSize: '1.25rem',
          color: 'var(--text-secondary)',
          maxWidth: '720px',
          margin: '0 auto 40px auto',
          fontWeight: 400,
          lineHeight: 1.6
        }}>
          Type-safe validation, schema enforcement, secret leak scanning, and instant IDE diagnostics. Sub-10ms performance. Offline-first. Zero cloud dependency.
        </p>

        {/* Terminal Copy Badge */}
        <div style={{
          maxWidth: '560px',
          margin: '0 auto 48px auto',
          background: 'var(--code-bg)',
          borderRadius: '16px',
          border: '1px solid var(--border)',
          padding: '16px 24px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          boxShadow: '0 20px 40px -10px rgba(15,23,42,0.25)'
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '14px' }}>
            <Terminal size={22} color="var(--accent-purple)" />
            <code style={{ fontSize: '1.05rem', color: '#38bdf8', fontWeight: 600 }}>
              $ npx razify check
            </code>
          </div>
          <button
            onClick={copyCommand}
            style={{
              background: 'rgba(255,255,255,0.08)',
              border: '1px solid rgba(255,255,255,0.15)',
              borderRadius: '8px',
              padding: '6px 14px',
              cursor: 'pointer',
              color: copied ? 'var(--accent-emerald)' : '#f8fafc',
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
        <div style={{ display: 'flex', justifyContent: 'center', gap: '16px', flexWrap: 'wrap', marginBottom: '56px' }}>
          <a href="#playground" className="btn-primary">
            Try Live Playground <ArrowRight size={18} />
          </a>
          <a href="https://marketplace.visualstudio.com" target="_blank" rel="noreferrer" className="btn-secondary">
            Get VS Code Extension
          </a>
        </div>

        {/* Stat Badges */}
        <div style={{
          display: 'flex',
          justifyContent: 'center',
          gap: '32px',
          flexWrap: 'wrap',
          borderTop: '1px solid var(--border)',
          paddingTop: '36px'
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <Zap size={20} color="var(--accent-purple)" />
            <span style={{ fontWeight: 700, fontSize: '0.95rem' }}>Sub-10ms Execution</span>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <Lock size={20} color="var(--accent-cyan)" />
            <span style={{ fontWeight: 700, fontSize: '0.95rem' }}>100% Offline & Private</span>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <Cpu size={20} color="var(--accent-emerald)" />
            <span style={{ fontWeight: 700, fontSize: '0.95rem' }}>Zero Cloud Lock-In</span>
          </div>
        </div>

      </div>
    </section>
  );
}
