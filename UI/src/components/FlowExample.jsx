import { useCallback, useState, useEffect, useRef } from "react";
import ReactFlow, {
  MiniMap,
  Controls,
  Background,
  useNodesState,
  useEdgesState,
  addEdge,
} from "reactflow";
import "reactflow/dist/style.css";

// Helper function to generate random position within viewport
const getRandomPosition = () => ({
  x: Math.random() * 500,
  y: Math.random() * 500,
});

// Helper function to generate random node type
const getRandomNodeType = () => {
  const types = ["input", "default", "output"];
  return types[Math.floor(Math.random() * types.length)];
};

function FlowExample() {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [nodeCount, setNodeCount] = useState(0);
  const [isSpawning, setIsSpawning] = useState(false);
  const [spawnInterval, setSpawnInterval] = useState(1000); // Default 1 second
  const spawnTimerRef = useRef(null);

  const onConnect = useCallback(
    (params) => setEdges((eds) => addEdge(params, eds)),
    [setEdges]
  );

  const addNewNode = useCallback(() => {
    const newNodeId = `node-${nodeCount + 1}`;
    const nodeType = getRandomNodeType();

    const newNode = {
      id: newNodeId,
      type: nodeType,
      data: {
        label: `${nodeType.charAt(0).toUpperCase() + nodeType.slice(1)} Node ${
          nodeCount + 1
        }`,
      },
      position: getRandomPosition(),
    };

    setNodes((nds) => [...nds, newNode]);
    setNodeCount((count) => count + 1);

    // If there are existing nodes, create a connection to a random existing node
    if (nodes.length > 0) {
      const randomExistingNode =
        nodes[Math.floor(Math.random() * nodes.length)];
      const newEdge = {
        id: `edge-${randomExistingNode.id}-${newNodeId}`,
        source: randomExistingNode.id,
        target: newNodeId,
      };
      setEdges((eds) => [...eds, newEdge]);
    }
  }, [nodes, nodeCount, setNodes, setEdges]);

  // Effect to handle automatic spawning
  useEffect(() => {
    if (isSpawning) {
      spawnTimerRef.current = setInterval(addNewNode, spawnInterval);
    } else {
      if (spawnTimerRef.current) {
        clearInterval(spawnTimerRef.current);
        spawnTimerRef.current = null;
      }
    }

    return () => {
      if (spawnTimerRef.current) {
        clearInterval(spawnTimerRef.current);
      }
    };
  }, [isSpawning, spawnInterval, addNewNode]);

  const toggleSpawning = () => {
    setIsSpawning((prev) => !prev);
  };

  const handleIntervalChange = (e) => {
    const newInterval = Math.max(100, parseInt(e.target.value) || 1000); // Minimum 100ms
    setSpawnInterval(newInterval);
  };

  return (
    <div style={{ width: "100vw", height: "100vh", position: "relative" }}>
      <div
        style={{
          position: "absolute",
          top: 10,
          left: 10,
          zIndex: 4,
          backgroundColor: "rgba(255, 255, 255, 0.9)",
          padding: "15px",
          borderRadius: "8px",
          boxShadow: "0 2px 4px rgba(0,0,0,0.1)",
        }}
      >
        <div style={{ marginBottom: "10px" }}>
          <button
            onClick={toggleSpawning}
            style={{
              padding: "10px 20px",
              fontSize: "16px",
              backgroundColor: isSpawning ? "#dc3545" : "#1a192b",
              color: "white",
              border: "none",
              borderRadius: "4px",
              cursor: "pointer",
              boxShadow: "0 2px 4px rgba(0,0,0,0.2)",
              marginRight: "10px",
            }}
          >
            {isSpawning ? "Stop Spawning" : "Start Spawning"}
          </button>
          <button
            onClick={addNewNode}
            style={{
              padding: "10px 20px",
              fontSize: "16px",
              backgroundColor: "#1a192b",
              color: "white",
              border: "none",
              borderRadius: "4px",
              cursor: "pointer",
              boxShadow: "0 2px 4px rgba(0,0,0,0.2)",
            }}
          >
            Add Single Node
          </button>
        </div>
        <div style={{ marginBottom: "10px" }}>
          <label style={{ marginRight: "10px" }}>
            Spawn Interval (ms):
            <input
              type="number"
              value={spawnInterval}
              onChange={handleIntervalChange}
              min="100"
              max="10000"
              style={{
                marginLeft: "5px",
                padding: "5px",
                width: "80px",
                borderRadius: "4px",
                border: "1px solid #ccc",
              }}
            />
          </label>
        </div>
        <div style={{ color: "#1a192b" }}>Total Nodes: {nodeCount}</div>
      </div>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        fitView
      >
        <Controls />
        <MiniMap />
        <Background variant="dots" gap={12} size={1} />
      </ReactFlow>
    </div>
  );
}

export default FlowExample;
