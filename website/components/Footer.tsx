'use client';

import React from 'react';
import { Github, Heart } from 'lucide-react';

export default function Footer() {
  return (
    <footer style={{
      borderTop: '1px solid var(--border)',
      background: 'var(--bg-secondary)',
      padding: '40px 0',
      fontSize: '0.9rem',
      color: 'var(--text-secondary)'
    }}>
      <div className="container" style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        flexWrap: 'wrap',
        gap: '20px'
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <img src="/logo.png" alt="Razify Logo" width="24" height="24" />
          <span style={{ fontWeight: 700, color: 'var(--text-primary)' }}>Razify</span>
          <span>— The Configuration Integrity Engine</span>
        </div>

        <div style={{ display: 'flex', gap: '20px' }}>
          <a href="https://github.com/Hossiy21/razify" target="_blank" rel="noreferrer" style={{ color: 'var(--text-secondary)', textDecoration: 'none' }}>GitHub</a>
          <a href="https://www.npmjs.com/package/razify" target="_blank" rel="noreferrer" style={{ color: 'var(--text-secondary)', textDecoration: 'none' }}>npm</a>
          <a href="https://marketplace.visualstudio.com" target="_blank" rel="noreferrer" style={{ color: 'var(--text-secondary)', textDecoration: 'none' }}>VS Code Marketplace</a>
          <a href="https://github.com/Hossiy21/razify/blob/main/LICENSE" target="_blank" rel="noreferrer" style={{ color: 'var(--text-secondary)', textDecoration: 'none' }}>MIT License</a>
        </div>

        <div>
          Built with <Heart size={14} color="var(--accent-rose)" fill="var(--accent-rose)" style={{ display: 'inline' }} /> for open source developers.
        </div>
      </div>
    </footer>
  );
}
