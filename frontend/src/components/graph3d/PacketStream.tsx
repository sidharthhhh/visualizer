import { useRef, useMemo } from 'react';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';

interface PacketProps {
  curve: THREE.Curve<THREE.Vector3>;
  speed?: number;
  color?: string;
  size?: number;
}

function Packet({ curve, speed = 1, color = '#00ff88', size = 0.03 }: PacketProps) {
  const meshRef = useRef<THREE.Mesh>(null);
  const progressRef = useRef(Math.random());

  useFrame((_, delta) => {
    if (meshRef.current) {
      progressRef.current = (progressRef.current + delta * speed * 0.5) % 1;
      const point = curve.getPointAt(progressRef.current);
      meshRef.current.position.copy(point);
    }
  });

  return (
    <mesh ref={meshRef}>
      <sphereGeometry args={[size, 8, 8]} />
      <meshBasicMaterial
        color={color}
        transparent
        opacity={0.9}
      />
      <pointLight color={color} intensity={0.5} distance={0.5} />
    </mesh>
  );
}

interface PacketStreamProps {
  start: [number, number, number];
  end: [number, number, number];
  protocol?: string;
  bandwidth?: number;
  packetCount?: number;
}

export default function PacketStream({
  start,
  end,
  protocol = 'tcp',
  bandwidth = 0,
  packetCount = 3,
}: PacketStreamProps) {
  const curve = useMemo(() => {
    const startVec = new THREE.Vector3(...start);
    const endVec = new THREE.Vector3(...end);
    const mid = new THREE.Vector3().lerpVectors(startVec, endVec, 0.5);
    mid.y += 0.5;
    return new THREE.QuadraticBezierCurve3(startVec, mid, endVec);
  }, [start, end]);

  const color = protocol === 'tcp' ? '#00d4ff' : '#a855f7';
  const speed = 0.5 + bandwidth * 0.01;
  const count = Math.min(Math.max(packetCount, 1), 10);

  return (
    <group>
      {Array.from({ length: count }, (_, i) => (
        <Packet
          key={i}
          curve={curve}
          speed={speed}
          color={color}
          size={0.02 + bandwidth * 0.001}
        />
      ))}
    </group>
  );
}
