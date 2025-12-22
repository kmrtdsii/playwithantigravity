import { useCallback, useEffect } from 'react';
import {
    ReactFlow,
    Background,
    Controls,
    MiniMap,
    useNodesState,
    useEdgesState,
    type Node,
    type Edge,
    Position,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import dagre from 'dagre';
import type { ObjectNode } from '../types/gitTypes';

interface ObjectGraphProps {
    commitId: string;
    objects: Record<string, ObjectNode>;
    onClose: () => void;
}

const nodeWidth = 172;
const nodeHeight = 36;

const getLayoutedElements = (nodes: Node[], edges: Edge[]) => {
    const dagreGraph = new dagre.graphlib.Graph();
    dagreGraph.setDefaultEdgeLabel(() => ({}));

    dagreGraph.setGraph({ rankdir: 'TB' });

    nodes.forEach((node) => {
        dagreGraph.setNode(node.id, { width: nodeWidth, height: nodeHeight });
    });

    edges.forEach((edge) => {
        dagreGraph.setEdge(edge.source, edge.target);
    });

    dagre.layout(dagreGraph);

    const layoutedNodes = nodes.map((node) => {
        const nodeWithPosition = dagreGraph.node(node.id);
        return {
            ...node,
            targetPosition: Position.Top,
            sourcePosition: Position.Bottom,
            position: {
                x: nodeWithPosition.x - nodeWidth / 2,
                y: nodeWithPosition.y - nodeHeight / 2,
            },
        };
    });

    return { nodes: layoutedNodes, edges };
};

const ObjectGraph: React.FC<ObjectGraphProps> = ({ commitId, objects, onClose }) => {
    const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
    const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

    // Build the graph from the commitId
    const buildGraph = useCallback(() => {
        if (!objects || !objects[commitId]) return { nodes: [], edges: [] };

        const newNodes: Node[] = [];
        const newEdges: Edge[] = [];
        const visited = new Set<string>();
        const queue = [commitId];

        while (queue.length > 0) {
            const currentId = queue.shift()!;
            if (visited.has(currentId)) continue;
            visited.add(currentId);

            const obj = objects[currentId];
            if (!obj) continue;

            let label = obj.id.substring(0, 6);
            let style = {};

            if (obj.type === 'commit') {
                label = `Commit: ${obj.id.substring(0, 6)}`;
                style = { background: '#f0f4ff', border: '1px solid #1a73e8', borderRadius: '5px' };
                if (obj.treeId) {
                    // Add edge to tree
                    newEdges.push({
                        id: `${currentId}-${obj.treeId}`,
                        source: currentId,
                        target: obj.treeId,
                        animated: true,
                    });
                    if (!visited.has(obj.treeId)) {
                        queue.push(obj.treeId);
                    }
                }
            } else if (obj.type === 'tree') {
                label = `Tree: ${obj.id.substring(0, 6)}`;
                style = { background: '#fffbeb', border: '1px solid #f59e0b', borderRadius: '5px' };
                if (obj.entries) {
                    obj.entries.forEach(entry => {
                        // Add edge
                        newEdges.push({
                            id: `${currentId}-${entry.hash}`,
                            source: currentId,
                            target: entry.hash,
                            animated: true,
                        });
                        // Add to queue
                        if (!visited.has(entry.hash)) {
                            queue.push(entry.hash);
                        }
                    });
                }
            } else if (obj.type === 'blob') {
                label = `Blob: ${obj.id.substring(0, 6)}`;
                style = { background: '#f3f4f6', border: '1px solid #9ca3af', borderRadius: '5px' };
            }

            newNodes.push({
                id: currentId,
                data: { label: <div><strong>{obj.type.toUpperCase()}</strong><br />{label}</div> },
                position: { x: 0, y: 0 }, // will be set by dagre
                style: { ...style, width: 150, fontSize: '12px' },
                type: 'default',
            });
        }
        return { nodes: newNodes, edges: newEdges };
    }, [commitId, objects]);

    // FIX: The commit lookup issue.
    // The `objects` map contains raw objects. The initial `commitId` passed is a commit hash.
    // The traversal needs to know how to go from Commit -> Tree.
    // My previous backend code in `state.go`:
    // case plumbing.CommitObject: node.Type = "commit"; node.Message = c.Message
    // It MISSES the tree hash in ObjectNode!
    // I should fix the backend to include TreeHash in ObjectNode for commits, OR rely on the main `commits` list.
    // Relying on `commits` list is easier since I already requested `TreeID` in `Commit` struct.
    // But `ObjectGraph` only takes `objects` and `commitId`.
    // I should probably pass `treeId` as prop or look it up.

    // Let's assume I fix the backend to populate `ObjectNode` with `TreeID` for commits too, or use `entries` for it? 
    // No, `entries` is for Tree.

    // Simplest fix: Add `treeId` to `ObjectNode` for commits in backend and TS.
    // Or pass the root Tree ID to this component instead of Commit ID?
    // "X-Ray" usually implies seeing inside the commit. So seeing "Commit -> Tree" link is valuable.

    // I will generate this file assuming `ObjectNode` handles it or I handle it here.
    // Let's modify the backend to include `TreeHASH` in ObjectNode for commits.

    useEffect(() => {
        const { nodes: builtNodes, edges: builtEdges } = buildGraph();
        const { nodes: layoutedNodes, edges: layoutedEdges } = getLayoutedElements(builtNodes, builtEdges);
        setNodes(layoutedNodes);
        setEdges(layoutedEdges);
    }, [buildGraph, setNodes, setEdges]);


    return (
        <div style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, zIndex: 1000, background: 'rgba(0,0,0,0.8)', display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
            <div style={{ width: '90%', height: '90%', background: 'white', borderRadius: '8px', overflow: 'hidden', position: 'relative' }}>
                <button onClick={onClose} style={{ position: 'absolute', top: 10, right: 10, zIndex: 10, padding: '5px 10px' }}>Close</button>
                <ReactFlow
                    nodes={nodes}
                    edges={edges}
                    onNodesChange={onNodesChange}
                    onEdgesChange={onEdgesChange}
                    fitView
                >
                    <Controls />
                    <MiniMap />
                    <Background gap={12} size={1} />
                </ReactFlow>
            </div>
        </div>
    );
};

export default ObjectGraph;
