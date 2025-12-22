import { useState } from 'react';
import { useGit } from '../../context/GitAPIContext';
import './BranchingStrategies.css';

const BranchingStrategies = () => {
    const { strategies, isSandbox, isForking, enterSandbox } = useGit();
    const [selectedId, setSelectedId] = useState<string | null>(null);

    const selectedStrategy = strategies.find(s => s.id === selectedId);

    return (
        <div className="strategies-container">
            <div className="strategies-sidebar">
                <h3>Select a Strategy</h3>
                <div className="strategy-list">
                    {strategies.map(s => (
                        <button
                            key={s.id}
                            className={`strategy-item ${selectedId === s.id ? 'active' : ''}`}
                            onClick={() => setSelectedId(s.id)}
                        >
                            {s.name}
                        </button>
                    ))}
                </div>
            </div>

            <div className="strategy-detail">
                {selectedStrategy ? (
                    <div className="strategy-content">
                        <h2>{selectedStrategy.name}</h2>
                        <p className="description">{selectedStrategy.description}</p>

                        <div className="flow-steps">
                            <h3>Workflow Steps:</h3>
                            <ul>
                                {selectedStrategy.flowSteps.map((step: string, i: number) => (
                                    <li key={i}>{step}</li>
                                ))}
                            </ul>
                        </div>

                        {!isSandbox && (
                            <div className="suggested-action">
                                <p>Want to try this strategy without affecting your work?</p>
                                <button
                                    onClick={enterSandbox}
                                    className="sandbox-btn"
                                    disabled={isForking}
                                >
                                    {isForking ? 'PREPARING SANDBOX...' : 'TRY IN SANDBOX MODE'}
                                </button>
                            </div>
                        )}

                        {isSandbox && (
                            <div className="sandbox-notice">
                                <p>You are in <b>Sandbox Mode</b>. Commands you run here are experimental.</p>
                            </div>
                        )}
                    </div>
                ) : (
                    <div className="no-selection">
                        <span style={{ fontSize: '48px' }}>🎓</span>
                        <p>Select a branching strategy to learn more about how it works.</p>
                    </div>
                )}
            </div>
        </div>
    );
};

export default BranchingStrategies;
