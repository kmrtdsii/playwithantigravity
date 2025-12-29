import React, { useState, useRef, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { SKILL_TREE, type SkillCommand } from '../../data/skillTree';
import { Modal } from '../common';
import { useMission } from '../../context/MissionContext';

interface SkillRadarProps {
    isOpen: boolean;
    onClose: () => void;
}

const SkillRadar: React.FC<SkillRadarProps> = ({ isOpen, onClose }) => {
    const { startMission } = useMission();
    const [selectedCommand, setSelectedCommand] = useState<SkillCommand | null>(null);
    const [hoveredCommand, setHoveredCommand] = useState<string | null>(null);
    const containerRef = useRef<HTMLDivElement>(null);
    const [svgSize, setSvgSize] = useState(500);

    // Resize handling
    useEffect(() => {
        const container = containerRef.current;
        if (!container) return;

        const updateSize = () => {
            const rect = container.getBoundingClientRect();
            // Maintain padding
            const newSize = Math.min(rect.width, rect.height) - 40;
            setSvgSize(Math.max(300, newSize));
        };

        updateSize();
        const resizeObserver = new ResizeObserver(updateSize);
        resizeObserver.observe(container);
        return () => resizeObserver.disconnect();
    }, [isOpen]);

    // === GEOMETRY CONSTANTS (ECCENTRIC LAYOUT) ===
    // The radar is defined by circles that conceptually share a left tangent.
    // This creates a "thick" side on the right where the labels go, and a thin side on the left.

    // Base Unit calculation
    // Max width needed consists of the largest circle diameter + padding.
    // In eccentric layout with shared left tangent at x=0:
    // Max X = Cx_max + R_max = R_max + R_max = 2*R_max.
    // So svgSize = 2 * R_max + Padding.
    const PADDING = 40;
    const MAX_D = svgSize - PADDING;

    // Ratios from image (approx 1.0 : 1.7 : 2.4 : 3.1)
    const MAX_RATIO = 3.1;
    const UNIT = MAX_D / (2 * MAX_RATIO);

    const R0 = UNIT * 1.0; // Basic
    const R1 = UNIT * 1.7; // Intermediate
    const R2 = UNIT * 2.4; // Proficient
    const R3 = UNIT * 3.1; // Advanced

    const RADII = [R0, R1, R2, R3];

    // Colors (Inner -> Outer)
    // Monotone / Slate Palette (Premium Grayscale)
    const RING_COLORS = [
        '#f8fafc', // Basic: Slate-50
        '#e2e8f0', // Intermediate: Slate-200
        '#94a3b8', // Proficient: Slate-400
        '#475569', // Advanced: Slate-600
    ];

    // Center Geometry
    // We want the circles to touch (or nearly touch) at the Left Tangent.
    // Let's define the Left Tangent X coordinate.
    const START_X = PADDING / 2;
    const CENTER_Y = svgSize / 2;

    // Center X of each circle:
    // If they share left tangent at START_X: Cx = START_X + R
    const getCx = (levelIndex: number) => START_X + RADII[levelIndex];



    // Title Wedge Angles (Degrees)
    // 0 is East (Right). Wedge is centered there.
    const TITLE_WEDGE_HALF = 10;
    const TITLE_START = -TITLE_WEDGE_HALF;
    const TITLE_END = TITLE_WEDGE_HALF;

    // Helper to distribute commands
    // Basic (Level 0): +60 to +320 degrees (Total 260 deg arc, wrapping around Back/Left)
    // Others: [-120, +120] degrees (Front/Right arc only)
    const getCommandSectors = (levelIndex: number, commands: SkillCommand[]) => {
        if (commands.length === 0) return [];

        // Special Case: Basic (Level 0)
        // User requested: +60 .. +320 degrees.
        if (levelIndex === 0) {
            const startLimit = 60;
            const endLimit = 320;
            const span = endLimit - startLimit;
            const step = span / commands.length;

            return commands.map((cmd, i) => ({
                command: cmd,
                startAngle: startLimit + i * step,
                endAngle: startLimit + (i + 1) * step
            }));
        }

        // Other Levels: Constrain to Front (Right Side)
        const SAFE_LIMIT = 120;

        // Split into Top and Bottom to respect the "Title Wedge" in the middle (0 deg).
        // Angle ranges:
        // Top: -SAFE_LIMIT to TITLE_START (-120 to -10)
        // Bottom: TITLE_END to SAFE_LIMIT (10 to 120)

        const half = Math.ceil(commands.length / 2);
        // Interleave or split? 
        // Let's split 50/50 for balance.
        const topCmds = commands.slice(0, half);
        const botCmds = commands.slice(half);

        const sectors: { command: SkillCommand; startAngle: number; endAngle: number }[] = [];

        // Bottom (Positive Angles)
        if (botCmds.length > 0) {
            const span = (SAFE_LIMIT - TITLE_END);
            const step = span / botCmds.length;
            botCmds.forEach((cmd, i) => {
                const s = TITLE_END + i * step;
                const e = TITLE_END + (i + 1) * step;
                sectors.push({ command: cmd, startAngle: s, endAngle: e });
            });
        }

        // Top (Negative Angles)
        if (topCmds.length > 0) {
            const span = (Math.abs(-SAFE_LIMIT - TITLE_START));
            const step = span / topCmds.length;
            topCmds.forEach((cmd, i) => {
                // Fill from 0 upwards
                const s = TITLE_START - i * step;
                const e = TITLE_START - (i + 1) * step;
                // e.g. -10, -30.
                sectors.push({ command: cmd, startAngle: e, endAngle: s });
            });
        }

        return sectors;
    };

    // Helper to generate Path for Eccentric Sector
    const describeEccentricSector = (levelIndex: number, startDeg: number, endDeg: number) => {
        const outerR = RADII[levelIndex];
        const outerCx = getCx(levelIndex);

        const innerR = levelIndex === 0 ? 0 : RADII[levelIndex - 1];
        const innerCx = levelIndex === 0 ? outerCx : getCx(levelIndex - 1);

        const radStart = (startDeg * Math.PI) / 180;
        const radEnd = (endDeg * Math.PI) / 180;

        // Points
        const osx = outerCx + outerR * Math.cos(radStart);
        const osy = CENTER_Y + outerR * Math.sin(radStart);
        const oex = outerCx + outerR * Math.cos(radEnd);
        const oey = CENTER_Y + outerR * Math.sin(radEnd);

        const isx = innerCx + innerR * Math.cos(radStart);
        const isy = CENTER_Y + innerR * Math.sin(radStart);
        const iex = innerCx + innerR * Math.cos(radEnd);
        const iey = CENTER_Y + innerR * Math.sin(radEnd);

        // Arc flags
        // We draw: InnerStart -> Line -> OuterStart -> Arc -> OuterEnd -> Line -> InnerEnd -> Arc -> InnerStart?
        // Or: InnerStart -> OuterStart (No, that's a cross)
        // Sector Loop:
        // 1. Move to Inner Start
        // 2. Line to Outer Start
        // 3. Arc to Outer End
        // 4. Line to Inner End
        // 5. Arc to Inner Start

        // Wait, standard order:
        // Start Angle is usually the "smaller" angle?
        // Let's assume text direction: Clockwise?
        // My sector generation produces logic like [10, 30].
        // So Start=10, End=30.
        // We want to draw 10 -> 30.

        const largeArc = Math.abs(endDeg - startDeg) > 180 ? 1 : 0;

        // Note: For Top hemisphere (negative angles), e.g. -30 to -10.
        // We ensure start < end so logic holds.

        // Path:
        // M isx isy
        // L osx osy
        // A (outer) 0 largeArc 1 oex oey  (Sweep 1 = Clockwise)
        // L iex iey
        // A (inner) 0 largeArc 0 isx isy  (Sweep 0 = Counter-Clockwise to return)

        // Wait, if start=-30 and end=-10. Clockwise goes -30 -> -10. Correct.

        if (levelIndex === 0) {
            // Pie slice
            return [
                `M ${innerCx} ${CENTER_Y}`,
                `L ${osx} ${osy}`,
                `A ${outerR} ${outerR} 0 ${largeArc} 1 ${oex} ${oey}`,
                `Z`
            ].join(' ');
        }

        return [
            `M ${isx} ${isy}`,
            `L ${osx} ${osy}`,
            `A ${outerR} ${outerR} 0 ${largeArc} 1 ${oex} ${oey}`,
            `L ${iex} ${iey}`,
            `A ${innerR} ${innerR} 0 ${largeArc} 0 ${isx} ${isy}`,
            `Z`
        ].join(' ');
    };

    const getOptimalFontSize = (levelIndex: number) => {
        // Larger fonts for outer rings
        return Math.max(11, svgSize * 0.015 + (levelIndex * 1.5));
    };

    return (
        <Modal isOpen={isOpen} onClose={onClose} title="Git Skill Radar" size="fullscreen" resizable disableBackdropClose>
            <div
                ref={containerRef}
                style={{
                    width: '100%',
                    height: '100%',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    background: 'linear-gradient(135deg, #f8fafc 0%, #e2e8f0 100%)',
                    borderRadius: '8px',
                    overflow: 'hidden'
                }}
            >
                <div style={{ width: svgSize, height: svgSize, position: 'relative' }}>
                    <svg
                        width={svgSize}
                        height={svgSize}
                        viewBox={`0 0 ${svgSize} ${svgSize}`}
                        style={{ overflow: 'visible' }}
                    >
                        <defs>
                            <filter id="shadow-outer">
                                <feDropShadow dx="0" dy="4" stdDeviation="8" floodColor="#000" floodOpacity="0.15" />
                            </filter>
                        </defs>

                        {/* RINGS BACKGROUND */}
                        {[...SKILL_TREE].reverse().map((level, i) => {
                            const levelIndex = SKILL_TREE.length - 1 - i;
                            const r = RADII[levelIndex];
                            const cx = getCx(levelIndex);

                            return (
                                <circle
                                    key={`ring-${level.id}`}
                                    cx={cx}
                                    cy={CENTER_Y}
                                    r={r}
                                    fill={RING_COLORS[levelIndex]}
                                    stroke="#fff"
                                    strokeWidth="2"
                                    filter="url(#shadow-outer)"
                                />
                            );
                        })}

                        {/* COMMAND SECTORS */}
                        {SKILL_TREE.map((level, levelIndex) => {
                            const sectors = getCommandSectors(levelIndex, level.commands);
                            return sectors.map(({ command, startAngle, endAngle }) => {
                                const isHovered = hoveredCommand === command.id;

                                // Text Position: Average of 4 corners of the sector
                                const outerCx = getCx(levelIndex);
                                const outerR = RADII[levelIndex];
                                const innerCx = levelIndex === 0 ? outerCx : getCx(levelIndex - 1);
                                const innerR = levelIndex === 0 ? 0 : RADII[levelIndex - 1];

                                const midAngle = (startAngle + endAngle) / 2;
                                const rad = midAngle * Math.PI / 180;

                                // Text Radius Adjustment
                                // Nudge Level 0 text outwards (ratio 0.65) to prevent crowding at the center.
                                const ratio = levelIndex === 0 ? 0.65 : 0.5;

                                const pxOut = outerCx + outerR * Math.cos(rad);
                                const pyOut = CENTER_Y + outerR * Math.sin(rad);

                                const pxIn = innerCx + innerR * Math.cos(rad);
                                const pyIn = CENTER_Y + innerR * Math.sin(rad);

                                const textX = pxIn + (pxOut - pxIn) * ratio;
                                const textY = pyIn + (pyOut - pyIn) * ratio;

                                const textColor = levelIndex >= 2 ? '#fff' : '#1e293b';

                                const isDisabled = command.disabled;

                                return (
                                    <g
                                        key={command.id}
                                        onClick={() => !isDisabled && setSelectedCommand(command)}
                                        onMouseEnter={() => !isDisabled && setHoveredCommand(command.id)}
                                        onMouseLeave={() => !isDisabled && setHoveredCommand(null)}
                                        style={{
                                            cursor: isDisabled ? 'not-allowed' : 'pointer',
                                            transition: 'all 0.3s ease',
                                            opacity: isDisabled ? 0.5 : 1
                                        }}
                                    >
                                        <path
                                            d={describeEccentricSector(levelIndex, startAngle, endAngle)}
                                            fill="transparent"
                                            stroke="rgba(255,255,255,0.1)"
                                            strokeWidth="1"
                                            className="hover-sector"
                                        />
                                        <motion.path
                                            d={describeEccentricSector(levelIndex, startAngle, endAngle)}
                                            fill={levelIndex >= 2 ? "rgba(255,255,255,0.15)" : "rgba(255,255,255,0.3)"}
                                            initial={{ opacity: 0 }}
                                            animate={{ opacity: isHovered ? 1 : 0 }}
                                            transition={{ duration: 0.15 }}
                                            stroke="none"
                                        />

                                        <text
                                            x={textX}
                                            y={textY}
                                            textAnchor="middle"
                                            dominantBaseline="middle"
                                            fill={textColor}
                                            fontSize={getOptimalFontSize(levelIndex)}
                                            fontWeight={isHovered ? 700 : 500}
                                            style={{ pointerEvents: 'none', userSelect: 'none', textShadow: levelIndex >= 2 ? '0 1px 2px rgba(0,0,0,0.3)' : 'none' }}
                                        >
                                            {isDisabled ? '🔒 ' : ''}{command.name.replace('git ', '')}
                                        </text>
                                    </g>
                                );
                            });
                        })}

                        {/* TITLE WEDGE (The "Beam") */}
                        {SKILL_TREE.map((level, levelIndex) => {
                            // Calculate centered position for title in the "Right Wedge"
                            // Midpoint between Inner and Outer arcs at 0 degrees
                            const cxOut = getCx(levelIndex);
                            const rOut = RADII[levelIndex];

                            const cxIn = levelIndex === 0 ? cxOut : getCx(levelIndex - 1);
                            const rIn = levelIndex === 0 ? 0 : RADII[levelIndex - 1];

                            // At 0 degrees (East):
                            // P_out = Cx_out + R_out
                            // P_in = Cx_in + R_in
                            // P_mid = (P_out + P_in) / 2
                            // But wait, the title should be shifted slightly to align nicely.

                            const textX = ((cxOut + rOut) + (cxIn + rIn)) / 2;

                            return (
                                <g key={`title-${level.id}`}>
                                    <path
                                        d={describeEccentricSector(levelIndex, TITLE_START, TITLE_END)}
                                        fill="rgba(255,255,255,0.9)"
                                        stroke="none"
                                    />
                                    <text
                                        x={textX}
                                        y={CENTER_Y}
                                        textAnchor="middle"
                                        dominantBaseline="middle"
                                        fontSize={Math.max(11, svgSize * 0.02)}
                                        fontWeight="800"
                                        fill="#0f172a"
                                    >
                                        {level.name.replace('Git ', '')}
                                    </text>
                                </g>
                            );
                        })}
                    </svg>
                </div>

                {/* MISSION CONTROL POPUP */}
                <AnimatePresence>
                    {selectedCommand && (
                        <motion.div
                            initial={{ opacity: 0, scale: 0.95, y: 20 }}
                            animate={{ opacity: 1, scale: 1, y: 0 }}
                            exit={{ opacity: 0, scale: 0.95, y: 20 }}
                            transition={{ duration: 0.2 }}
                            style={{
                                position: 'fixed',
                                bottom: '40px',
                                right: '40px',
                                width: '380px',
                                background: 'rgba(255, 255, 255, 0.95)',
                                backdropFilter: 'blur(12px)',
                                padding: '24px',
                                borderRadius: '20px',
                                boxShadow: '0 20px 50px rgba(0,0,0,0.15), 0 0 0 1px rgba(255,255,255,0.5)',
                                border: '1px solid rgba(0,0,0,0.05)',
                                zIndex: 1000
                            }}
                        >
                            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'start', marginBottom: '16px' }}>
                                <h2 style={{ margin: 0, fontSize: '24px', fontWeight: 800, color: '#1e293b' }}>
                                    {selectedCommand.name}
                                </h2>
                                <button
                                    onClick={() => setSelectedCommand(null)}
                                    style={{
                                        background: '#f1f5f9',
                                        border: 'none',
                                        borderRadius: '50%',
                                        width: '32px',
                                        height: '32px',
                                        display: 'flex',
                                        alignItems: 'center',
                                        justifyContent: 'center',
                                        cursor: 'pointer',
                                        color: '#64748b'
                                    }}
                                >
                                    ✕
                                </button>
                            </div>

                            <div style={{ marginBottom: '24px' }}>
                                <span style={{
                                    display: 'inline-block',
                                    padding: '4px 12px',
                                    borderRadius: '100px',
                                    background: '#dbeafe',
                                    color: '#2563eb',
                                    fontSize: '12px',
                                    fontWeight: 700,
                                    marginBottom: '12px'
                                }}>
                                    MISSION BRIEF
                                </span>
                                <p style={{ margin: 0, color: '#475569', lineHeight: 1.6, fontSize: '15px' }}>
                                    {selectedCommand.description}
                                </p>
                            </div>

                            <button
                                style={{
                                    width: '100%',
                                    padding: '16px',
                                    background: 'linear-gradient(135deg, #3b82f6 0%, #2563eb 100%)',
                                    color: 'white',
                                    border: 'none',
                                    borderRadius: '12px',
                                    fontSize: '16px',
                                    fontWeight: 700,
                                    cursor: 'pointer',
                                    boxShadow: '0 4px 12px rgba(37, 99, 235, 0.3)',
                                    display: 'flex',
                                    justifyContent: 'center',
                                    alignItems: 'center',
                                    gap: '8px'
                                }}
                                onClick={() => {
                                    if (selectedCommand && selectedCommand.missionId) {
                                        startMission(selectedCommand.missionId);
                                        onClose();
                                    } else {
                                        alert("Mission not yet implemented for this skill.");
                                    }
                                }}
                            >
                                <span>Start Practice</span>
                                <span>→</span>
                            </button>
                        </motion.div>
                    )}
                </AnimatePresence>
            </div>
        </Modal >
    );
};

export default SkillRadar;
