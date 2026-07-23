'use client';

import React, { useState, useEffect } from 'react';
import { Sun, Moon, Github, Sparkles } from 'lucide-react';

export default function Navbar() {
  const [theme, setTheme] = useState<'light' | 'dark'>('light');

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
  }, [theme]);

  const toggleTheme = () => {
    setTheme((prev) => (prev === 'light' ? 'dark' : 'light'));
  };

  return (
    <header style={{
      position: 'sticky',
      top: 0,
      zIndex: 50,
      backdropFilter: 'blur(16px)',
      borderBottom: '1px solid var(--border)',
      background: 'var(--bg-card)'
    }}>
      <div className="container" style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        height: '72px'
      }}>
        {/* Brand Logo */}
        <a href="#" style={{ display: 'flex', alignItems: 'center', gap: '12px', textDecoration: 'none' }}>
          <img src="/logo.png" alt="Razify Logo" width="36" height="36" style={{ borderRadius: '10px' }} />
          <span style={{ fontSize: '1.3rem', fontWeight: 800, color: 'var(--text-primary)', letterSpacing: '-0.02em' }}>
            Razify
          </span>
          <span style={{
            fontSize: '0.75rem',
            padding: '3px 10px',
            borderRadius: '20px',
            background: 'var(--glow-purple)',
            color: 'var(--accent-purple)',
            fontWeight: 700,
            border: '1px solid var(--border-glow)'
          }}>
            v1.0.1
          </span>
        </a>

        {/* Nav Links */}
        <nav style={{ display: 'flex', gap: '28px', alignItems: 'center' }}>
          <a href="#features" style={{ color: 'var(--text-secondary)', textDecoration: 'none', fontWeight: 600, fontSize: '0.95rem' }}>Features</a>
          <a href="#playground" style={{ color: 'var(--text-secondary)', textDecoration: 'none', fontWeight: 600, fontSize: '0.95rem' }}>Playground</a>
          <a href="#integrations" style={{ color: 'var(--text-secondary)', textDecoration: 'none', fontWeight: 600, fontSize: '0.95rem' }}>Integrations</a>
          <a href="#sponsors" style={{ color: 'var(--text-secondary)', textDecoration: 'none', fontWeight: 600, fontSize: '0.95rem' }}>Sponsors</a>
        </nav>

        {/* Actions */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          {/* Theme Toggle Button */}
          <button
            onClick={toggleTheme}
            title={`Switch to ${theme === 'light' ? 'Dark' : 'Light'} Mode`}
            style={{
              background: 'var(--bg-secondary)',
              border: '1px solid var(--border)',
              borderRadius: '10px',
              padding: '8px 14px',
              cursor: 'pointer',
              color: 'var(--text-primary)',
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              fontSize: '0.85rem',
              fontWeight: 600
            }}
          >
            {theme === 'light' ? <Moon size={16} color="#6366f1" /> : <Sun size={16} color="#f59e0b" />}
            {theme === 'light' ? 'Dark' : 'Light'}
          </button>

          {/* GitHub Repository Link */}
          <a
            href="https://github.com/Hossiy21/razify"
            target="_blank"
            rel="noreferrer"
            className="btn-secondary"
            style={{ padding: '8px 16px', fontSize: '0.85rem' }}
          >
            <Github size={18} />
            GitHub
          </a>
        </div>
      </div>
    </header>
  );
}
