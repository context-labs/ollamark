import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import axios from 'axios';
import { FaTwitter } from 'react-icons/fa';
import { DiWindows, DiApple, DiLinux } from "react-icons/di";

const BenchmarkDetail = () => {
  const { submissionid } = useParams();
  const [benchmark, setBenchmark] = useState(null);

  useEffect(() => {
    axios.get(`/api/benchmark/${submissionid}`)
      .then(response => setBenchmark(response.data))
      .catch(error => console.error("Error fetching benchmark:", error));
  }, [submissionid]);

  if (!benchmark) return <div className="text-gray-400">Loading...</div>;

  const tweetText = `Benchmarked ${benchmark.model_name} at ${benchmark.tokens_per_second.toFixed(2)} tok/sec on ${benchmark.sys_info.os} ${benchmark.gpu_info ? `with a ${benchmark.gpu_info.name}` : ''}\n\nhttps://ollamark.com/marks/${benchmark.submission_id}`;
  const tweetUrl = `https://twitter.com/intent/tweet?text=${encodeURIComponent(tweetText)}`;

  return (
    <div className="bg-black p-11 md:p-11 shadow-md text-white text-center" style={{borderRadius: "35px"}}>
      <h1 className="text-3xl md:text-3xl font-bold mb-4">Details of {benchmark.model_name} ollamark</h1>
      <div className="mb-8">
        <p className="text-6xl md:text-9xl font-bold text-blue-500">{benchmark.tokens_per_second.toFixed(2)}</p>
        <p className="text-lg md:text-xl text-gray-400">Ollamark Score (Tokens Per Second)</p>
        <p className="font-semibold text-xl">{benchmark.submission_id}</p>
      </div>
      <hr className="border-gray-800"/><br />
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div>
          <p className="text-lg mb-2 text-gray-400">Benchmarks</p>
          <p className="font-semibold text-xl">{benchmark.iterations}</p>
        </div>
        <div>
          <p className="text-lg mb-2 text-gray-400">Timestamp</p>
          <p className="font-semibold text-xl">{new Date(benchmark.timestamp * 1000).toLocaleString()}</p>
        </div>
        <div>
          <p className="text-lg mb-2 text-gray-400">Ollama Version</p>
          <p className="font-semibold text-xl text-white">{benchmark.ollama_version}</p>
        </div>
      </div>
      <div className="col-span-2 flex justify-center my-4">
        <p className="font-semibold text-xl">
          {benchmark.sys_info.os.includes("darwin") ? <DiApple style={{fontSize: "5em"}} /> : 
           benchmark.sys_info.os.includes("windows") ? <DiWindows style={{fontSize: "5em"}} /> : 
           <DiLinux style={{fontSize: "5em"}} />}
        </p>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <p className="text-lg mb-2 text-gray-400">Arch</p>
          <p className="font-semibold text-xl">{benchmark.sys_info.arch}</p>
        </div>
        <div>
          <p className="text-lg mb-2 text-gray-400">Kernel Version</p>
          <p className="font-semibold text-xl">{benchmark.sys_info.kernel}</p>
        </div>
        <div>
          <p className="text-lg mb-2 text-gray-400">Client Type</p>
          <p className="font-semibold text-xl">{benchmark.client_type}</p>
        </div>
        <div>
          <p className="text-lg mb-2 text-gray-400">Client Version</p>
          <p className="font-semibold text-xl">{benchmark.client_version}</p>
        </div>
        <div>
          <p className="text-lg mb-2 text-gray-400">CPU</p>
          <p className="font-semibold text-xl">{benchmark.sys_info.cpu_name}</p>
        </div>
        <div>
          <p className="text-lg mb-2 text-gray-400">RAM</p>
          <p className="font-semibold text-xl">{benchmark.sys_info.memory}</p>
        </div>
        {benchmark.gpu_info && (
          <>
            <div>
              <p className="text-lg mb-2 text-gray-400">GPU</p>
              <p className="font-semibold text-xl">{benchmark.gpu_info.name}</p>
            </div>
            <div>
              <p className="text-lg mb-2 text-gray-400">GPU Driver Version</p>
              <p className="font-semibold text-xl">{benchmark.gpu_info.driver_version}</p>
            </div>
          </>
        )}
      </div>
      <br /><hr className="border-gray-800"/><br />
      <a href={tweetUrl} target="_blank" rel="noopener noreferrer" className="inline-flex items-center px-4 py-2 bg-blue-500 text-white font-semibold text-lg rounded-lg shadow-md hover:bg-blue-600">
        <FaTwitter className="mr-2" /> Share on Twitter
      </a>
      <br />
    </div>
  );
};

export default BenchmarkDetail;