import React from "react";
import { SearchProvider, SearchBox, Results, Paging, Facet, Sorting, ResultsPerPage } from "@elastic/react-search-ui";
import ElasticsearchAPIConnector from "@elastic/search-ui-elasticsearch-connector";
import "@elastic/react-search-ui-views/lib/styles/styles.css";

const connector = new ElasticsearchAPIConnector({
  host: "http://localhost:5173/search", // Proxied to Go backend via Vite
  index: "torrents"
});

const config = {
  apiConnector: connector,
  searchQuery: {
    // Fields the user is allowed to search against
    search_fields: {
      title: {},
      infohash: {},
      "files.Path": {},
      category: {}
    },
    // Fields Elasticsearch should return to React
    result_fields: {
      title: { raw: {} },
      infohash: { raw: {} },
      size: { raw: {} },
      category: { raw: {} },
      discovered_on: { raw: {} }
    },
    facets: {
      category: { type: "value", size: 30 }
    }
  }
};

const formatSize = (bytes) => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

const TorrentCard = ({ result }) => {
  const title = result.title?.raw || "Unknown";
  const sizeBytes = result.size?.raw || 0;
  const size = formatSize(sizeBytes);
  const categories = Array.isArray(result.category?.raw) 
      ? result.category.raw.join(", ") 
      : result.category?.raw || "None";
  const infohash = result.infohash?.raw || "N/A";
  
  return (
    <div className="card w-full bg-base-100 shadow-xl mb-4 border border-base-300">
      <div className="card-body">
        {/* Render the torrent title */}
        <h2 className="card-title text-primary">{title}</h2>
        
        {/* Render metadata */}
        <div className="flex gap-4 text-sm text-gray-500 mt-2">
          <span><strong className="text-base-content">Size:</strong> {size}</span>
          <span><strong className="text-base-content">Category:</strong> {categories}</span>
        </div>
        
        <p className="text-xs font-mono text-gray-400 mt-2 truncate">
          Hash: {infohash}
        </p>

        {/* DaisyUI Action Buttons */}
        <div className="card-actions justify-end mt-4">
          <a 
            href={`magnet:?xt=urn:btih:${infohash}&dn=${encodeURIComponent(title)}`}
            className="btn btn-sm btn-secondary"
          >
            Magnet Link
          </a>
        </div>
      </div>
    </div>
  );
};

export default function App() {
  return (
    <SearchProvider config={config}>
      <div className="min-h-screen bg-base-200 p-4 md:p-8">
        <h1 className="text-4xl font-bold mb-8 text-center text-primary">
          Decentralized Torrent Indexer
        </h1>

        <div className="max-w-6xl mx-auto flex flex-col md:flex-row gap-8">
          {/* Sidebar for Facets */}
          <div className="w-full md:w-1/4">
            <div className="bg-base-100 p-4 rounded-box shadow-xl border border-base-300 sticky top-4">
              <h3 className="font-bold text-lg mb-4 text-secondary">Filters</h3>
              <Facet field="category" label="Category" />
            </div>
          </div>

          {/* Main Results */}
          <div className="w-full md:w-3/4">
            {/* Search Bar Input */}
            <div className="mb-8">
              <SearchBox 
                autocompleteMinimumCharacters={3}
                autocompleteResults={{ title: {} }}
                inputProps={{ className: "input input-bordered input-primary w-full text-lg shadow-sm", placeholder: "Search for torrents (name, hash, or files)..." }}
              />
            </div>

            <div className="flex justify-between items-center mb-4 bg-base-100 p-4 rounded-box shadow-xl border border-base-300">
              <div className="w-1/2 md:w-1/3">
                <Sorting
                  label={"Sort by"}
                  sortOptions={[
                    { name: "Relevance", value: "", direction: "" },
                    { name: "Newest", value: "discovered_on", direction: "desc" },
                    { name: "Oldest", value: "discovered_on", direction: "asc" }
                  ]}
                />
              </div>
              <div className="w-1/2 md:w-1/3 flex justify-end items-center gap-2">
                <span className="text-sm font-semibold">Results per page</span>
                <ResultsPerPage
                  options={[10, 20, 50]}
                />
              </div>
            </div>

            {/* Render Results using the DaisyUI Card */}
            <div className="space-y-4">
              <Results resultView={TorrentCard} titleField="title" urlField="infohash" />
            </div>

            {/* Pagination Controls */}
            <div className="mt-8 flex justify-center bg-base-100 p-4 rounded-box shadow-xl border border-base-300">
              <Paging />
            </div>
          </div>
        </div>
      </div>
    </SearchProvider>
  );
}
