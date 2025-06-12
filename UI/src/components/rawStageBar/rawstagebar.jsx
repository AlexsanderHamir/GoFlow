import { ResponsiveBar } from "@nivo/bar";
import "./rawstagebar.css";

const CustomTooltip = ({ id, value, indexValue, color }) => (
  <div
    style={{
      padding: "12px 16px",
      background: "rgba(26, 26, 31, 0.95)",
      border: `2px solid ${color}`,
      borderRadius: 12,
      color: "#fff",
      fontWeight: 500,
      fontSize: "1rem",
      boxShadow: "0 8px 32px rgba(0,0,0,0.4)",
      minWidth: 140,
      backdropFilter: "blur(10px)",
    }}
  >
    <div style={{ marginBottom: 6, fontSize: "0.9rem", opacity: 0.8 }}>
      {indexValue}
    </div>
    <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
      <div
        style={{
          width: 8,
          height: 8,
          borderRadius: "50%",
          backgroundColor: color,
        }}
      />
      <span style={{ color: "#e2e8f7" }}>{id}</span>
      <span style={{ color, fontWeight: 600, marginLeft: 4 }}>
        {value.toLocaleString()}
      </span>
    </div>
  </div>
);

const StageBar = ({ stageData }) => {
  // Enhanced color scheme for better professional appearance
  const customColors = [
    "#646cff", // Primary blue
    "#9333ea", // Purple
    "#06b6d4", // Cyan
    "#10b981", // Green
    "#f59e0b", // Amber
    "#ef4444", // Red
    "#8b5cf6", // Violet
  ];

  return (
    <div className="stage-bar-chart">
      <ResponsiveBar
        data={stageData}
        keys={[
          "output_items",
          "dropped_items",
          "throughput",
          "drop_rate",
          "processed_items",
        ]}
        indexBy="stage_name"
        groupMode="grouped"
        valueScale={{ type: "linear" }}
        colors={customColors}
        borderColor={{ from: "color", modifiers: [["darker", 0.8]] }}
        borderWidth={0}
        borderRadius={6}
        labelSkipWidth={12}
        labelSkipHeight={12}
        labelTextColor={{ from: "color", modifiers: [["darker", 2]] }}
        legends={[
          {
            dataFrom: "keys",
            anchor: "bottom-right",
            direction: "column",
            justify: false,
            translateX: 110,
            translateY: 0,
            itemsSpacing: 6,
            itemWidth: 100,
            itemHeight: 22,
            itemDirection: "left-to-right",
            itemOpacity: 0.9,
            symbolSize: 16,
            symbolShape: "circle",
            itemTextColor: "#c4c7d0",
            effects: [
              {
                on: "hover",
                style: {
                  itemOpacity: 1,
                  itemTextColor: "#e2e8f7",
                },
              },
            ],
          },
        ]}
        axisTop={null}
        axisRight={null}
        axisBottom={{
          tickSize: 6,
          tickPadding: 8,
          tickRotation: -35,
          legend: "Stage Name",
          legendPosition: "middle",
          legendOffset: 50,
          format: (value) =>
            value.length > 12 ? `${value.substring(0, 12)}...` : value,
        }}
        axisLeft={{
          tickSize: 6,
          tickPadding: 8,
          tickRotation: 0,
          legend: "Metrics Value",
          legendPosition: "middle",
          legendOffset: -50,
          format: (value) => {
            if (value >= 1000000) return `${(value / 1000000).toFixed(1)}M`;
            if (value >= 1000) return `${(value / 1000).toFixed(1)}K`;
            return value.toLocaleString();
          },
        }}
        // Proper centering with balanced margins
        margin={{ top: 40, right: 140, bottom: 80, left: 80 }}
        padding={0.25}
        innerPadding={2}
        enableGridX={false}
        enableGridY={true}
        gridYValues={5}
        enableLabel={true}
        labelFormat={(value) => {
          if (value >= 1000000) return `${(value / 1000000).toFixed(1)}M`;
          if (value >= 1000) return `${(value / 1000).toFixed(1)}K`;
          return value.toLocaleString();
        }}
        animate={true}
        motionConfig="gentle"
        theme={{
          background: "transparent",
          text: {
            fontSize: 12,
            fill: "#c4c7d0",
            fontWeight: 500,
          },
          axis: {
            domain: {
              line: {
                stroke: "#555",
                strokeWidth: 2,
              },
            },
            ticks: {
              line: {
                stroke: "#555",
                strokeWidth: 1,
              },
              text: {
                fill: "#c4c7d0",
                fontSize: 11,
                fontWeight: 500,
              },
            },
            legend: {
              text: {
                fill: "#e2e8f7",
                fontSize: 13,
                fontWeight: 600,
              },
            },
          },
          grid: {
            line: {
              stroke: "rgba(255,255,255,0.08)",
              strokeWidth: 1,
              strokeDasharray: "2 4",
            },
          },
          legends: {
            text: {
              fill: "#c4c7d0",
              fontSize: 12,
              fontWeight: 500,
            },
          },
          tooltip: {
            container: {
              background: "rgba(26, 26, 31, 0.95)",
              color: "#fff",
              fontSize: "12px",
              borderRadius: "8px",
              boxShadow: "0 8px 32px rgba(0,0,0,0.4)",
              border: "1px solid rgba(255,255,255,0.1)",
            },
          },
        }}
        tooltip={CustomTooltip}
        role="application"
        ariaLabel="Stage performance metrics bar chart"
      />
    </div>
  );
};

export default StageBar;
