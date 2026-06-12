import React from "react";
import "./globals.css";

export const metadata = {
  title: "SkyFee - Bitcoin Lightning Tuition Settlement",
  description: "Real-time institutional school-fee payments over Bitcoin Lightning and Safaricom M-Pesa",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className="bg-slate-50 text-slate-900 font-sans antialiased">
        <nav className="border-b border-slate-200 bg-white/90 backdrop-blur-md px-6 py-4 flex justify-between items-center shadow-sm sticky top-0 z-40">
          <div className="flex items-center space-x-2">
            <span className="text-xl font-extrabold tracking-tight text-blue-600">
              ⚡ SkyFee
            </span>
            <span className="text-xs uppercase bg-blue-50 border border-blue-100 tracking-wider text-blue-700 px-2 py-0.5 rounded font-semibold font-mono">
              v1.0
            </span>
          </div>
          <div className="flex space-x-6 text-sm font-semibold text-slate-600">
            <a href="/dashboard" className="hover:text-blue-600 transition-colors duration-200">
              Dashboard
            </a>
            <a href="/register" className="hover:text-blue-600 transition-colors duration-200">
              Register Institution
            </a>
          </div>
        </nav>
        <main className="min-h-screen bg-slate-50">{children}</main>
      </body>
    </html>
  );
}
