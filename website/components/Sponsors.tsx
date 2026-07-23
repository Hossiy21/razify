'use client';

import React from 'react';
import { Heart, ShieldCheck, Zap, Star, ExternalLink, Building2, User, Users } from 'lucide-react';

export default function Sponsors() {
  return (
    <section id="sponsors" style={{ padding: '80px 0', borderTop: '1px solid var(--border)' }}>
      <div className="container">
        
        {/* Section Header */}
        <div style={{ textAlign: 'center', marginBottom: '56px' }}>
          <div style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: '8px',
            padding: '6px 16px',
            borderRadius: '20px',
            background: 'rgba(244, 63, 94, 0.1)',
            border: '1px solid rgba(244, 63, 94, 0.3)',
            color: 'var(--accent-rose)',
            fontWeight: 600,
            fontSize: '0.85rem',
            marginBottom: '16px'
          }}>
            <Heart size={16} fill="var(--accent-rose)" />
            Open Source Sustainability
          </div>
          <h2 style={{ fontSize: '2.5rem', fontWeight: 800, marginBottom: '12px' }}>
            Sponsor <span className="gradient-text">Razify</span>
          </h2>
          <p style={{ color: 'var(--text-secondary)', fontSize: '1.1rem', maxWidth: '650px', margin: '0 auto' }}>
            Razify is 100% free, MIT-licensed, and independent. Your sponsorship ensures sub-10ms performance, active maintainer support, and ecosystem growth.
          </p>
        </div>

        {/* Sponsorship Tiers Grid */}
        <div style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))',
          gap: '24px',
          marginBottom: '56px'
        }}>

          {/* Tier 1: Supporter */}
          <div className="glass-card" style={{ padding: '32px', display: 'flex', flexDirection: 'column', justifyContent: 'space-between' }}>
            <div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '12px' }}>
                <User color="var(--accent-cyan)" size={24} />
                <h3 style={{ fontSize: '1.25rem', fontWeight: 700 }}>Individual Supporter</h3>
              </div>
              <div style={{ fontSize: '2rem', fontWeight: 800, color: 'var(--text-primary)', marginBottom: '16px' }}>
                $5 <span style={{ fontSize: '0.9rem', color: 'var(--text-muted)', fontWeight: 400 }}>/ month</span>
              </div>
              <ul style={{ listStyle: 'none', display: 'flex', flexDirection: 'column', gap: '10px', color: 'var(--text-secondary)', fontSize: '0.9rem' }}>
                <li style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <ShieldCheck size={16} color="var(--accent-emerald)" /> Sponsor badge on GitHub
                </li>
                <li style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <ShieldCheck size={16} color="var(--accent-emerald)" /> Shoutout in Release Notes
                </li>
                <li style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <ShieldCheck size={16} color="var(--accent-emerald)" /> Backer Discord role
                </li>
              </ul>
            </div>
            <a
              href="https://github.com/sponsors/Hossiy21"
              target="_blank"
              rel="noreferrer"
              className="btn-secondary"
              style={{ marginTop: '24px', justifyContent: 'center' }}
            >
              Become a Supporter <ExternalLink size={16} />
            </a>
          </div>

          {/* Tier 2: Team Backer */}
          <div className="glass-card" style={{
            padding: '32px',
            display: 'flex',
            flexDirection: 'column',
            justifyContent: 'space-between',
            border: '2px solid var(--accent-purple)',
            position: 'relative'
          }}>
            <div style={{
              position: 'absolute',
              top: '-12px',
              right: '24px',
              background: 'var(--accent-purple)',
              color: '#ffffff',
              fontSize: '0.75rem',
              fontWeight: 700,
              padding: '2px 12px',
              borderRadius: '12px'
            }}>
              MOST POPULAR
            </div>
            <div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '12px' }}>
                <Users color="var(--accent-purple)" size={24} />
                <h3 style={{ fontSize: '1.25rem', fontWeight: 700 }}>Team / Startup Backer</h3>
              </div>
              <div style={{ fontSize: '2rem', fontWeight: 800, color: 'var(--text-primary)', marginBottom: '16px' }}>
                $49 <span style={{ fontSize: '0.9rem', color: 'var(--text-muted)', fontWeight: 400 }}>/ month</span>
              </div>
              <ul style={{ listStyle: 'none', display: 'flex', flexDirection: 'column', gap: '10px', color: 'var(--text-secondary)', fontSize: '0.9rem' }}>
                <li style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <ShieldCheck size={16} color="var(--accent-emerald)" /> Logo on GitHub README
                </li>
                <li style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <ShieldCheck size={16} color="var(--accent-emerald)" /> Logo on Documentation Website
                </li>
                <li style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <ShieldCheck size={16} color="var(--accent-emerald)" /> Priority GitHub issue triage
                </li>
              </ul>
            </div>
            <a
              href="https://github.com/sponsors/Hossiy21"
              target="_blank"
              rel="noreferrer"
              className="btn-primary"
              style={{ marginTop: '24px', justifyContent: 'center' }}
            >
              Sponsor as a Team <ExternalLink size={16} />
            </a>
          </div>

          {/* Tier 3: Enterprise Sponsor */}
          <div className="glass-card" style={{ padding: '32px', display: 'flex', flexDirection: 'column', justifyContent: 'space-between' }}>
            <div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '12px' }}>
                <Building2 color="var(--accent-rose)" size={24} />
                <h3 style={{ fontSize: '1.25rem', fontWeight: 700 }}>Enterprise Sponsor</h3>
              </div>
              <div style={{ fontSize: '2rem', fontWeight: 800, color: 'var(--text-primary)', marginBottom: '16px' }}>
                $249 <span style={{ fontSize: '0.9rem', color: 'var(--text-muted)', fontWeight: 400 }}>/ month</span>
              </div>
              <ul style={{ listStyle: 'none', display: 'flex', flexDirection: 'column', gap: '10px', color: 'var(--text-secondary)', fontSize: '0.9rem' }}>
                <li style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <ShieldCheck size={16} color="var(--accent-emerald)" /> Large logo placement on homepage
                </li>
                <li style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <ShieldCheck size={16} color="var(--accent-emerald)" /> Dedicated support channel SLA
                </li>
                <li style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <ShieldCheck size={16} color="var(--accent-emerald)" /> Custom validation rule requests
                </li>
              </ul>
            </div>
            <a
              href="https://github.com/sponsors/Hossiy21"
              target="_blank"
              rel="noreferrer"
              className="btn-secondary"
              style={{ marginTop: '24px', justifyContent: 'center' }}
            >
              Enterprise Sponsorship <ExternalLink size={16} />
            </a>
          </div>

        </div>

      </div>
    </section>
  );
}
