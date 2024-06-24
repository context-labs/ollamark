// By Carsen Klock 2024 under the MIT license
// https://github.com/context-labs/ollamark
// https://ollamark.com
// Ollamark Web Client

import React from 'react';
import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
import BenchmarkTable from './BenchmarkTable';
import BenchmarkDetail from './BenchmarkDetail';
import Download from './Download';
import Footer from './Footer';
import Header from './Header';
import './App.css';

function App() {
  return (
    <Router>
      <div className="min-h-screen flex flex-col items-center bg-black-100">
<Header />

        <main className="container mx-auto py-6 px-4 flex-grow">
          <Routes>
            <Route path="/marks/:submissionid" element={<BenchmarkDetail />} />
            <Route path="/" element={<BenchmarkTable />} />
            <Route path="/download" element={<Download />} />
          </Routes>
        </main>
        <Footer />
      </div>
    </Router>
  );
}

export default App;