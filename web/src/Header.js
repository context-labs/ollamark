import React from 'react';
import logo from './logo.svg';
import { FaGithub, FaDownload, FaRegHandshake } from 'react-icons/fa';
import { RiExternalLinkLine } from "react-icons/ri";

const Header = () => {
  return (
    <header className="bg-black shadow w-full">
      <nav className="w-full text-white" style={{ backgroundColor: '#101216' }}>
        <div className="container mx-auto py-2 px-3 flex justify-between items-center">
          <a href="/download" className="text-sm font-semibold flex items-center">Download <FaDownload className="ml-2" /></a>
          <a href="https://github.com/context-labs/ollamark" target="_blank" rel="noreferrer" className="text-sm font-semibold flex items-center">Github <FaGithub className="ml-2" /></a>
          <a href="https://kuzco.xyz" className="text-sm font-semibold flex items-center" target="_blank" rel="noreferrer">Contribute <FaRegHandshake className="ml-2" /></a>
          <a href="https://ollama.com" target="_blank" rel="noreferrer" className="text-sm font-semibold flex items-center">Ollama <RiExternalLinkLine className="ml-2" /></a>
        </div>
      </nav>
      <div className="container mx-auto py-4 px-4 flex flex-col items-center">
        <a href="/"><img src={logo} alt="Ollamark.com" width="40px" className="mb-2" /></a>
        <h1 className="text-xl font-bold">Ollamark.com</h1>
        <p className="text-gray-500">Ollama AI Model Benchmarking</p>
      </div>
    </header>
  );
};

export default Header;