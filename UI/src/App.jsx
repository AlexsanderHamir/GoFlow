import { useState } from "react";
import reactLogo from "./assets/react.svg";
import viteLogo from "/vite.svg";
import "./App.css";

function App() {
  const [count, setCount] = useState(0);
  const [directoryFiles, setDirectoryFiles] = useState({
    goroutine: [],
    stages: [],
  });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState(null);

  const fetchDirectoryFiles = async (directory) => {
    try {
      const response = await fetch(`http://localhost:8080/${directory}`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const files = await response.json();
      return files;
    } catch (err) {
      throw new Error(`Failed to fetch ${directory}: ${err.message}`);
    }
  };

  const fetchAllFiles = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const [goroutineFiles, stagesFiles] = await Promise.all([
        fetchDirectoryFiles("goroutine"),
        fetchDirectoryFiles("stages"),
      ]);

      setDirectoryFiles({
        goroutine: goroutineFiles,
        stages: stagesFiles,
      });
    } catch (err) {
      setError(err.message);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      <div>
        <a href="https://vite.dev" target="_blank">
          <img src={viteLogo} className="logo" alt="Vite logo" />
        </a>
        <a href="https://react.dev" target="_blank">
          <img src={reactLogo} className="logo react" alt="React logo" />
        </a>
      </div>
      <h1>GoFlow File Viewer</h1>
      <div className="card">
        <button onClick={() => setCount((count) => count + 1)}>
          count is {count}
        </button>
        <button onClick={fetchAllFiles} disabled={isLoading}>
          {isLoading ? "Loading..." : "Fetch Files"}
        </button>
        {error && <p className="error">Error: {error}</p>}

        {(directoryFiles.goroutine.length > 0 ||
          directoryFiles.stages.length > 0) && (
          <div className="files-container">
            <div className="files-section">
              <h2>Goroutine Files:</h2>
              <ul>
                {directoryFiles.goroutine.map((file, index) => (
                  <li key={`goroutine-${index}`}>
                    <span className="file-name">{file}</span>
                    <button
                      className="view-button"
                      onClick={() =>
                        window.open(
                          `http://localhost:8080/goroutine/${file}`,
                          "_blank"
                        )
                      }
                    >
                      View
                    </button>
                  </li>
                ))}
              </ul>
            </div>

            <div className="files-section">
              <h2>Stages Files:</h2>
              <ul>
                {directoryFiles.stages.map((file, index) => (
                  <li key={`stages-${index}`}>
                    <span className="file-name">{file}</span>
                    <button
                      className="view-button"
                      onClick={() =>
                        window.open(
                          `http://localhost:8080/stages/${file}`,
                          "_blank"
                        )
                      }
                    >
                      View
                    </button>
                  </li>
                ))}
              </ul>
            </div>
          </div>
        )}
      </div>
    </>
  );
}

export default App;
