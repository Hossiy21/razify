import './globals.css';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Razify — The Configuration Integrity Engine',
  description: 'Type-safe environment variable validation, schema compliance, secret leak scanning, and sub-10ms performance.',
  keywords: ['razify', 'env', 'dotenv', 'environment variables', 'validation', 'config integrity', 'secret scanner', 'cli', 'vscode'],
  openGraph: {
    title: 'Razify — The Configuration Integrity Engine',
    description: 'Type-safe environment variable validation, schema compliance, and secret leak scanning.',
    images: ['/logo.png'],
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" data-theme="dark">
      <body>{children}</body>
    </html>
  );
}
