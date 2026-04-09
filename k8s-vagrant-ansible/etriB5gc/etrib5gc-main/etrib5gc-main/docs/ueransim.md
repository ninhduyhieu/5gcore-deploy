## Bugs:

 - Network release UE
 - UE Register again (Periodic Update) with PduSessionStatus having 1 active session
 - AMF leave that session suspended
 - UE think the session active but there is no bearer at gnB.

Correct behavior?

When application send uplink data at UE, UE should detect at RRC layer that there is no bearer. It then send ServiceRequest (MO-Data) with UplinkDataStatus -> AMF ask SMF to reactive the session.

