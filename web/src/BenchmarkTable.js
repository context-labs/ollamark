import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import axios from 'axios';
import { DiWindows, DiApple, DiLinux } from "react-icons/di";
import { FaSyncAlt } from "react-icons/fa";
import './style.css';

const BenchmarkTable = () => {
  const [benchmarks, setBenchmarks] = useState([]);
  const [totalBenchmarks, setTotalBenchmarks] = useState(0);
  const [allBenchmarks, setAllBenchmarks] = useState(0);
  const [sortBy, setSortBy] = useState('timestamp');
  const [order, setOrder] = useState('desc');
  const [modelFilter, setModelFilter] = useState('');
  const [ollamaVersionFilter, setOllamaVersionFilter] = useState('');
  const [osFilter, setOsFilter] = useState('');
  const [cpuFilter, setCpuFilter] = useState('');
  const [gpuFilter, setGpuFilter] = useState('');
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useState(10);
  const [modelOptions, setModelOptions] = useState([]);
  const [loading, setLoading] = useState(false);
  const [postLoadingSpin, setPostLoadingSpin] = useState(false);
  
  useEffect(() => {
    setPage(1); // Reset to first page when filters change
  }, [modelFilter, osFilter, cpuFilter, gpuFilter, ollamaVersionFilter]);

  useEffect(() => {
    // fetchBenchmarks every 10 seconds
    const interval = setInterval(() => {
      fetchBenchmarks();
      fetchAllBenchmarks();
    }, 10000);
    return () => clearInterval(interval);
  }, [sortBy, order, modelFilter, osFilter, cpuFilter, gpuFilter, ollamaVersionFilter, page, limit]);

  useEffect(() => {
    fetchBenchmarks();
    fetchAllBenchmarks();
    fetchModelOptions();
  }, [sortBy, order, modelFilter, osFilter, cpuFilter, gpuFilter, ollamaVersionFilter, page, limit]);

  const fetchBenchmarks = async () => {
    setLoading(true);
    try {
      const response = await axios.get('http://localhost:3333/api/benchmarks', {
        params: {
          sort_by: sortBy,
          order: order,
          model: modelFilter,
          os: osFilter,
          cpu: cpuFilter,
          gpu: gpuFilter,
          ollama_version: ollamaVersionFilter,
          page: page,
          limit: limit
        }
      });
      setBenchmarks(response.data.benchmarks || []);
    } catch (error) {
      console.error('Error fetching benchmarks:', error);
    } finally {
      setLoading(false);
      setPostLoadingSpin(true);
      setTimeout(() => setPostLoadingSpin(false), 500);
    }
  };

  const fetchAllBenchmarks = async () => {
    try {
      const response = await axios.get('http://localhost:3333/api/benchmarks', {
        params: {
          sort_by: "tokenspersecond",
          order: "desc",
          limit: 0 // Set limit to 0 to fetch all benchmarks
        }
      });
      setAllBenchmarks(response.data.benchmarks || []);
      setTotalBenchmarks(response.data.total || 0);
    } catch (error) {
      console.error('Error fetching all benchmarks:', error);
    }
  };

  const fetchModelOptions = async () => {
    const cachedOptions = localStorage.getItem('modelOptions');
    if (cachedOptions) {
      setModelOptions(JSON.parse(cachedOptions));
      return;
    }
    try {
      const response = await axios.get('http://localhost:3333/api/model-list');
      setModelOptions(response.data.models);
      localStorage.setItem('modelOptions', JSON.stringify(response.data.models));
    } catch (error) {
      console.error('Error fetching model options:', error);
    }
  };

  const handlePageChange = (newPage) => {
    if (newPage >= 1 && newPage <= Math.ceil(totalBenchmarks / limit)) {
      setPage(newPage);
    }
  };

  const getFastestModel = (minParams = 7, maxParams = 8) => {
    if (!Array.isArray(allBenchmarks) || allBenchmarks.length === 0) return null;
    const filteredBenchmarks = allBenchmarks.filter(benchmark => {
      const modelInfo = modelOptions.find(model => model.Name === benchmark.model_name);
      if (!modelInfo) return false;
      const params = parseFloat(modelInfo.Parameters.replace('B', ''));
      return params >= minParams && params <= maxParams;
    });
    if (filteredBenchmarks.length === 0) return null;
    return filteredBenchmarks.reduce((prev, current) => (prev.tokens_per_second > current.tokens_per_second) ? prev : current);
  };

  const getTopBenchmarkedModel = () => {
    if (!Array.isArray(allBenchmarks) || allBenchmarks.length === 0) return null;
    const modelCount = allBenchmarks.reduce((acc, benchmark) => {
      acc[benchmark.model_name] = (acc[benchmark.model_name] || 0) + 1;
      return acc;
    }, {});
    const topModel = Object.keys(modelCount).reduce((a, b) => modelCount[a] > modelCount[b] ? a : b);
    return topModel;
  };

  const getFastestModelByOS = (os, minParams = 7, maxParams = 8) => {
    console.log('All Benchmarks:', allBenchmarks.length);
    if (!Array.isArray(allBenchmarks) || allBenchmarks.length === 0) return null;
    const filteredBenchmarks = allBenchmarks.filter(benchmark => {
      console.log('Benchmark OS:', benchmark.sys_info.os);
      const modelInfo = modelOptions.find(model => model.Name === benchmark.model_name);
      if (!modelInfo) return false;
      const params = parseFloat(modelInfo.Parameters.replace('B', ''));
      return benchmark.sys_info.os.toLowerCase() === os.toLowerCase() &&
             params >= minParams && params <= maxParams;
    });
    console.log('Filtered Benchmarks for', os, ':', filteredBenchmarks);
    if (filteredBenchmarks.length === 0) return null;
    return filteredBenchmarks.reduce((prev, current) => (prev.tokens_per_second > current.tokens_per_second) ? prev : current);
  };
  
  // Update the calls to getFastestModelByOS
  const fastestWindowsModel = getFastestModelByOS('windows', 7, 8);
  const fastestMacOSModel = getFastestModelByOS('darwin', 7, 8); 
  const fastestLinuxModel = getFastestModelByOS('linux', 7, 8);

  const getFastestGPUModel = (minParams = 7, maxParams = 8) => {
    if (!Array.isArray(allBenchmarks) || allBenchmarks.length === 0) return null;
    const filteredBenchmarks = allBenchmarks.filter(benchmark => {
      const modelInfo = modelOptions.find(model => model.Name === benchmark.model_name);
      if (!modelInfo) return false;
      const params = parseFloat(modelInfo.Parameters.replace('B', ''));
      return params >= minParams && params <= maxParams && benchmark.gpu_info && benchmark.gpu_info.name;
    });
    if (filteredBenchmarks.length === 0) return null;
    return filteredBenchmarks.reduce((prev, current) => (prev.tokens_per_second > current.tokens_per_second) ? prev : current);
  };
  
  // Usage
  const fastestGPUModel = getFastestGPUModel(7, 8);

  const fastestModel = getFastestModel(7, 8);
  const topBenchmarkedModel = getTopBenchmarkedModel();

  return (
    <div className="rounded">
            <dl class="grid grid-cols-1 gap-0.5 overflow-hidden rounded-2xl text-center sm:grid-cols-2 lg:grid-cols-3">
        <div class="flex flex-col topgrid p-8">
          <dt class="text-sm font-semibold leading-6 text-gray-500">Fastest 7-8B Model on Windows</dt>
          <dd class="order-first text-3xl font-semibold tracking-tight text-white">{fastestWindowsModel ? fastestWindowsModel.model_name : '...'}</dd>
        </div>
        <div class="flex flex-col topgrid p-8">
          <dt class="text-sm font-semibold leading-6 text-gray-500">Fastest 7-8B Model on MacOS</dt>
          <dd class="order-first text-3xl font-semibold tracking-tight text-white">{fastestMacOSModel ? fastestMacOSModel.model_name : '...'}</dd>
        </div>
        <div class="flex flex-col topgrid p-8">
          <dt class="text-sm font-semibold leading-6 text-gray-500">Fastest 7-8B Model on Linux</dt>
          <dd class="order-first text-3xl font-semibold tracking-tight text-white">{fastestLinuxModel ? fastestLinuxModel.model_name : '...'}</dd>
        </div>

        <div class="flex flex-col topgrid p-8">
          <dt class="text-sm font-semibold leading-6 text-gray-500">Total Ollamarks</dt>
          <dd class="order-first text-3xl font-semibold tracking-tight text-white">{totalBenchmarks ? totalBenchmarks : '0'}</dd>
        </div>
        {/* <div class="flex flex-col topgrid p-8">
          <dt class="text-sm font-semibold leading-6 text-gray-500">Overall Fastest Model</dt>
          <dd class="order-first text-3xl font-semibold tracking-tight text-white">{fastestModel ? fastestModel.model_name : '...'}</dd>
        </div> */}
        <div class="flex flex-col topgrid p-8">
          <dt class="text-sm font-semibold leading-6 text-gray-500">Fastest GPU on 7-8B Model</dt>
          <dd class="order-first text-3xl font-semibold tracking-tight text-white">
            {fastestGPUModel ? `${fastestGPUModel.gpu_info.name}` : '...'}
          </dd>
        </div>
        <div class="flex flex-col topgrid p-8">
          <dt class="text-sm font-semibold leading-6 text-gray-500">Most Benchmarked Model</dt>
          <dd class="order-first text-3xl font-semibold tracking-tight text-white">{topBenchmarkedModel ? topBenchmarkedModel : '...'}</dd>
        </div>
      </dl>
      <div className="filters mb-4 mt-5 flex flex-wrap gap-2">
        <select
          value={sortBy}
          onChange={(e) => setSortBy(e.target.value)}
          className="border p-2 flex-grow"
        >
          <option value="timestamp">Timestamp</option>
          <option value="tokenspersecond">Tokens Per Second</option>
        </select>
        <select
          value={order}
          onChange={(e) => setOrder(e.target.value)}
          className="border p-2 flex-grow"
        >
          <option value="desc">Descending</option>
          <option value="asc">Ascending</option>
        </select>
        <select
          value={modelFilter}
          onChange={(e) => setModelFilter(e.target.value)}
          className="border p-2 flex-grow"
        >
          <option value="">All Models</option>
          {modelOptions.map((model, index) => (
            <option key={index} value={model.Name}>
              {model.Name} ({model.Parameters}{model.Quantization ? `, ${model.Quantization}` : ''})
            </option>
          ))}
        </select>
        <select
          value={osFilter}
          onChange={(e) => setOsFilter(e.target.value)}
          className="border p-2 flex-grow"
        >
          <option value="">All OS</option>
          <option value="windows">Windows</option>
          <option value="darwin">macOS</option>
          <option value="linux">Linux</option>
        </select>
        <select
          value={ollamaVersionFilter}
          onChange={(e) => setOllamaVersionFilter(e.target.value)}
          className="border p-2 flex-grow"
        >
          <option value="">All Ollama Versions</option>
          <option value="0.1.46">0.1.46</option>
          <option value="0.1.45">0.1.45</option>
          <option value="0.1.44">0.1.44</option>
          <option value="0.1.43">0.1.43</option>
          <option value="0.1.42">0.1.42</option>
          <option value="0.1.41">0.1.41</option>
          <option value="0.1.40">0.1.40</option>
          <option value="0.1.39">0.1.39</option>
          <option value="0.1.38">0.1.38</option>
          <option value="0.1.37">0.1.37</option>
          <option value="0.1.36">0.1.36</option>
          <option value="0.1.35">0.1.35</option>
        </select>
        <input
          type="text"
          placeholder="Filter by CPU Name"
          value={cpuFilter}
          onChange={(e) => setCpuFilter(e.target.value)}
          className="border p-2 flex-grow"
        />
        <input
          type="text"
          placeholder="Filter by GPU Name"
          value={gpuFilter}
          onChange={(e) => setGpuFilter(e.target.value)}
          className="border p-2 flex-grow"
        />
        <button
          onClick={fetchBenchmarks}
          className="bg-gray rounded text-white flex items-center"
        >
          <FaSyncAlt className={`m-2 ${loading || postLoadingSpin ? 'animate-spin' : ''}`} />
          
        </button>
      </div>
      <div className="benchmark-table-container overflow-x-auto">
        <table className="benchmark-table w-full text-left border-collapse shadow">
          <thead>
            <tr>
              <th className="p-2 border-b">ID</th>
              <th className="p-2 border-b">Model</th>
              <th className="p-2 border-b">OS</th>
              <th className="p-2 border-b">CPU</th>
              <th className="p-2 border-b">GPU</th>
              <th className="p-2 border-b">Ollama Version</th>
              <th className="p-2 border-b">Tokens Per Second</th>
              <th className="p-2 border-b">Iterations</th>
              <th className="p-2 border-b">Timestamp</th>
            </tr>
          </thead>
          <tbody>
            {benchmarks.map((benchmark, index) => (
              <tr key={index} className="hover:bg-gray-100">
                <td className="p-2 border-b font-bold">  <Link to={`/marks/${benchmark.submission_id}`}>
    {benchmark.submission_id ? benchmark.submission_id.slice(0, 8) : 'N/A'}
  </Link></td>
                <td className="p-2 border-b font-bold">{benchmark.model_name}</td>
                <td className="os-icon p-2">{benchmark.sys_info.os === 'windows' ? <> <DiWindows className="mr-2" /> Windows</> : benchmark.sys_info.os === 'darwin' ? <><DiApple className="mr-2" /> Apple</> : <><DiLinux className="mr-2" /> Linux</>}</td>
                <td className="p-2 border-b">{benchmark.sys_info.cpu_name}</td>
                <td className="p-2 border-b">{benchmark.gpu_info.name}</td>
                <td className="p-2 border-b">{benchmark.ollama_version}</td>
                <td className="p-2 border-b">{benchmark.tokens_per_second.toFixed(2)}</td>
                <td className="p-2 border-b">{benchmark.iterations}</td>
                <td className="p-2 border-b">{new Date(benchmark.timestamp * 1000).toLocaleString()}</td>
              </tr>
            ))}
            {benchmarks.length === 0 && (
              <tr>
                <td colSpan="7" className="p-2 text-center">No Ollamarks found...</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      <div className="pagination flex justify-between items-center mt-4">
        <button
          onClick={() => handlePageChange(page - 1)}
          disabled={page === 1}
          className="p-2 bg-black rounded disabled:opacity-50"
        >
          Previous
        </button>
        <span className="p-2 text-center flex-grow">Page {page} of {Math.ceil(totalBenchmarks / limit)}</span>
        <button
          onClick={() => handlePageChange(page + 1)}
          disabled={page === Math.ceil(totalBenchmarks / limit)}
          className="p-2 bg-black rounded disabled:opacity-50"
        >
          Next
        </button>
      </div>
    </div>
  );
};

export default BenchmarkTable;