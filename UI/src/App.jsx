import { useState, useEffect } from "react";
import "./App.css";

function App() {
  const [goroutineFilesData, setGoroutineFilesData] = useState([]);
  const [stagesFilesData, setStagesFilesData] = useState([]);

  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState(null);

  const GetAllData = async () => {
    setIsLoading(true);
    setError(null);

    try {
      const [goroutineFiles, stagesFiles] = await Promise.all([
        fetchDirectoryFiles("goroutine"),
        fetchDirectoryFiles("stages"),
      ]);

      const filesNames = {
        goroutine: goroutineFiles,
        stages: stagesFiles,
      };

      await fetchAllFilesData(filesNames);
    } catch (err) {
      setError(err.message);
    } finally {
      setIsLoading(false);
    }
  };

  const fetchDirectoryFiles = async (directory) => {
    try {
      const response = await fetch(`/api/list?dir=${directory}`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const files = await response.json();
      return files;
    } catch (err) {
      throw new Error(`Failed to fetch ${directory}: ${err.message}`);
    }
  };

  const fetchAllFilesData = async (filesNames) => {
    const goroutineData = [];
    for (const routineInfo of filesNames.goroutine) {
      const data = await fetchFileData(routineInfo.path);
      goroutineData.push(data);
    }
    setGoroutineFilesData(goroutineData);

    const stageData = [];
    for (const stageInfo of filesNames.stages) {
      const data = await fetchFileData(stageInfo.path);
      stageData.push(data);
    }
    setStagesFilesData(stageData);
  };

  const fetchFileData = async (filePath) => {
    const response = await fetch(`/${filePath}`);
    const data = await response.json();
    return data;
  };

  useEffect(() => {
    if (goroutineFilesData.length > 0 && stagesFilesData.length > 0) {
      console.log("goroutineFilesData", goroutineFilesData);
      console.log("stagesFilesData", stagesFilesData);
    }
  }, [goroutineFilesData, stagesFilesData]);

  return (
    <>
      <h1>GoFlow File Viewer</h1>
      <div className="card">
        <button onClick={GetAllData} disabled={isLoading}>
          {isLoading ? "Loading..." : "Fetch Files"}
        </button>

        {error && <p className="error">Error: {error}</p>}
      </div>
    </>
  );
}

export default App;
