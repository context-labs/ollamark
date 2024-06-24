import React from 'react';
import { FaWindows, FaApple, FaLinux } from 'react-icons/fa';

const Download = () => {
  const version = "0.0.1"
  const downloads = [
    {
      name: 'Windows',
      icon: FaWindows,
      url: 'https://github.com/context-labs/ollamark/releases', 
      version: version,
    },
    {
      name: 'macOS',
      icon: FaApple,
      url: 'https://github.com/context-labs/ollamark/releases',
      version: version,
    },
    {
      name: 'Linux',
      icon: FaLinux,
      url: 'https://github.com/context-labs/ollamark/releases',
      version: version,
    },
  ];

  return (
    <div className="max-w-4xl mx-auto px-4 py-16">
      <h1 className="text-4xl font-bold text-center mb-12">Download Ollamark</h1>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
        {downloads.map((download) => (
          <div key={download.name} className="flex flex-col items-center">
            <download.icon className="text-6xl mb-4" />
            <h2 className="text-2xl font-semibold mb-2">{download.name}</h2>
            <p className="text-gray-600 mb-4">Version {download.version}</p>
            <a
              href={download.url}
              className="bg-blue-500 hover:bg-blue-600 text-white font-bold py-2 px-4 rounded"
              download
            >
              Download
            </a>
          </div>
        ))}
      </div>
    </div>
  );
};

export default Download;