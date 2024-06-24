import React from 'react';
import './App.css';
import logo from './logo.svg';

const Footer = () => {
  return (
<footer className="bg-black text-white text-center p-8 w-full">
          <div className="container mx-auto grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="flex flex-col items-center">
              <h2 className="text-lg font-bold">Ollamark.com</h2>
              <p className="text-gray-400">Ollama AI Model Benchmarking</p>
              <a href="/"><img src={logo} alt="Ollamark.com" width="80px" className="mt-4" /></a>
            </div>
            <div>
              <h2 className="text-lg font-bold">Links</h2>
              <ul className="text-gray-400">
                <li><a href="/download" className="hover:underline">Download Ollamark</a></li>
                <li><a href="https://github.com/context-labs/ollamark" target="_blank" rel="noreferrer" className="hover:underline">Ollamark Github</a></li>
                <li><a href="https://kuzco.xyz" target="_blank" rel="noreferrer" className="hover:underline">Contribute Compute</a></li>
                <li><a href="https://ollama.com" target="_blank" rel="noreferrer" className="hover:underline">Ollama</a></li>
              </ul>
            </div>
            <div>
              <h2 className="text-lg font-bold">Contact & Policies</h2>
              <p className="text-gray-400">Twitter: <a href="https://twitter.com/ollamark" target="_blank" rel="noreferrer" className="hover:underline">@ollamark</a></p>
              <p className="text-gray-400"><a href="https://ollamark.com/privacy" target="_blank" rel="noreferrer" className="hover:underline">Privacy Policy</a></p>
              <p className="text-gray-400"><a href="https://ollamark.com/tos" target="_blank" rel="noreferrer" className="hover:underline">Terms of Service</a></p>
            </div>
          </div>
          <div className="mt-8">
            <p>Made with 🦙🦙🦙 by <a href="https://twitter.com/carsenklock" className="hover:underline">Carsen Klock</a><br />🖥️ OSS with MIT license at <a href="https://github.com/context-labs/ollamark" className="hover:underline">GitHub</a></p>
          </div>
        </footer>
  );
};

export default Footer;
